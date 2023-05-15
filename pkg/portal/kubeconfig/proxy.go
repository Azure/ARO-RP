package kubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

const (
	kubeconfigTimeout = time.Hour
)

type contextKey int

const (
	contextKeyClient contextKey = iota
	contextKeyResponse
)

// director is called by the ReverseProxy.  It converts an incoming request into
// the one that'll go out to the API server.  It also resolves an HTTP client
// that will be able to make the ongoing request.
//
// Unfortunately the signature of httputil.ReverseProxy.Director does not allow
// us to return values.  We get around this limitation slightly naughtily by
// storing return information in the request context.
func (k *Kubeconfig) director(r *http.Request) {
	ctx := r.Context()

	portalDoc, _ := ctx.Value(middleware.ContextKeyPortalDoc).(*api.PortalDocument)
	if portalDoc == nil || portalDoc.Portal.Kubeconfig == nil {
		k.error(r, http.StatusForbidden, nil)
		return
	}

	resourceID := strings.Join(strings.Split(r.URL.Path, "/")[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) ||
		!strings.EqualFold(resourceID, portalDoc.Portal.ID) {
		k.error(r, http.StatusBadRequest, nil)
		return
	}

	key := struct {
		resourceID string
		elevated   bool
	}{
		resourceID: portalDoc.Portal.ID,
		elevated:   portalDoc.Portal.Kubeconfig.Elevated,
	}

	cli := k.clientCache.Get(key)
	if cli == nil {
		var err error
		cli, err = k.cli(ctx, key.resourceID, key.elevated)
		if err != nil {
			k.error(r, http.StatusInternalServerError, err)
			return
		}

		k.clientCache.Put(key, cli)
	}

	r.RequestURI = ""
	r.URL.Scheme = "https"
	r.URL.Host = "kubernetes:6443"
	r.URL.Path = "/" + strings.Join(strings.Split(r.URL.Path, "/")[11:], "/")
	r.Header.Del("Authorization")
	r.Host = r.URL.Host

	// http.Request.WithContext returns a copy of the original Request with the
	// new context, but we have no way to return it, so we overwrite our
	// existing request.
	*r = *r.WithContext(context.WithValue(ctx, contextKeyClient, cli))
}

// cli returns an appropriately configured HTTP client for forwarding the
// incoming request to a cluster
func (k *Kubeconfig) cli(ctx context.Context, resourceID string, elevated bool) (*http.Client, error) {
	openShiftDoc, err := k.dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	kc := openShiftDoc.OpenShiftCluster.Properties.AROSREKubeconfig
	if elevated {
		kc = openShiftDoc.OpenShiftCluster.Properties.AROServiceKubeconfig
	}

	if len(kc) == 0 {
		return nil, fmt.Errorf("kubeconfig is nil")
	}

	var kubeconfig *clientcmdv1.Config
	err = yaml.Unmarshal(kc, &kubeconfig)
	if err != nil {
		return nil, err
	}

	var b []byte
	b = append(b, kubeconfig.AuthInfos[0].AuthInfo.ClientKeyData...)
	b = append(b, kubeconfig.AuthInfos[0].AuthInfo.ClientCertificateData...)

	clientKey, clientCerts, err := utilpem.Parse(b)
	if err != nil {
		return nil, err
	}

	_, caCerts, err := utilpem.Parse(kubeconfig.Clusters[0].Cluster.CertificateAuthorityData)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	for _, caCert := range caCerts {
		pool.AddCert(caCert)
	}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: restconfig.DialContext(k.dialer, openShiftDoc.OpenShiftCluster),
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{
							clientCerts[0].Raw,
						},
						PrivateKey: clientKey,
					},
				},
				RootCAs: pool,
			},
		},
	}, nil
}

// roundTripper is called by ReverseProxy to make the onward request happen.  We
// check if we had an error earlier and return that if we did.  Otherwise we dig
// out the client and call it.
func (k *Kubeconfig) roundTripper(r *http.Request) (*http.Response, error) {
	if resp, ok := r.Context().Value(contextKeyResponse).(*http.Response); ok {
		return resp, nil
	}

	cli := r.Context().Value(contextKeyClient).(*http.Client)
	resp, err := cli.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusSwitchingProtocols {
		resp.Body = newCancelBody(resp.Body.(io.ReadWriteCloser), kubeconfigTimeout)
	}

	return resp, err
}

func (k *Kubeconfig) error(r *http.Request, statusCode int, err error) {
	if err != nil {
		k.Log.Warn(err)
	}

	w := responsewriter.New(r)
	http.Error(w, http.StatusText(statusCode), statusCode)

	*r = *r.WithContext(context.WithValue(r.Context(), contextKeyResponse, w.Response()))
}

// cancelBody is a workaround for the fact that http timeouts are incompatible
// with hijacked connections (https://github.com/golang/go/issues/31391):
// net/http.cancelTimerBody does not implement Writer.
type cancelBody struct {
	io.ReadWriteCloser
	t *time.Timer
	c chan struct{}
}

func (b *cancelBody) wait() {
	select {
	case <-b.t.C:
		b.ReadWriteCloser.Close()
	case <-b.c:
		b.t.Stop()
	}
}

func (b *cancelBody) Close() error {
	select {
	case b.c <- struct{}{}:
	default:
	}

	return b.ReadWriteCloser.Close()
}

func newCancelBody(rwc io.ReadWriteCloser, d time.Duration) io.ReadWriteCloser {
	b := &cancelBody{
		ReadWriteCloser: rwc,
		t:               time.NewTimer(d),
		c:               make(chan struct{}),
	}

	go b.wait()

	return b
}
