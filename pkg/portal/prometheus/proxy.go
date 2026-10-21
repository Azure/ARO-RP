package prometheus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"mime"
	"net"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/portforward"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// Unfortunately the signature of httputil.ReverseProxy.Director does not allow
// us to return errors.  We get around this limitation slightly naughtily by
// storing return information in the request context.

type contextKey int

const (
	contextKeyClient contextKey = iota
	contextKeyResponse
)

// Director modifies the request to point to the clusters prometheus instance
func (p *Prometheus) Director(r *http.Request) {
	ctx := r.Context()

	resourceID := strings.Join(strings.Split(r.URL.Path, "/")[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) {
		p.error(r, http.StatusBadRequest, nil)
		return
	}

	cli := p.clientCache.Get(resourceID)
	if cli == nil {
		var err error
		cli, err = p.Cli(ctx, resourceID)
		if err != nil {
			p.error(r, http.StatusInternalServerError, err)
			return
		}

		p.clientCache.Put(resourceID, cli)
	}

	r.RequestURI = ""
	r.URL.Host, r.URL.Scheme = p.GetPrometheusHostAndScheme()
	r.URL.Path = "/" + strings.Join(strings.Split(r.URL.Path, "/")[10:], "/")
	r.Header.Del("Cookie")
	r.Header.Del("Referer")
	r.Host = r.URL.Host

	// http.Request.WithContext returns a copy of the original Request with the
	// new context, but we have no way to return it, so we overwrite our
	// existing request.
	*r = *r.WithContext(context.WithValue(ctx, contextKeyClient, cli))
}

func (p *Prometheus) GetPrometheusHostAndScheme() (string, string) {
	return "prometheus-k8s-0:9090", "http"
}

func (p *Prometheus) Cli(ctx context.Context, resourceID string) (*http.Client, error) {
	openShiftDoc, err := p.dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	restconfig, err := restconfig.RestConfig(p.dialer, openShiftDoc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return portforward.DialContext(ctx, p.log, restconfig, "openshift-monitoring", "prometheus-k8s-0", "9090")
			},
		},
	}, nil
}

func (p *Prometheus) RoundTripper(r *http.Request) (*http.Response, error) {
	if resp, ok := r.Context().Value(contextKeyResponse).(*http.Response); ok {
		return resp, nil
	}

	cli := r.Context().Value(contextKeyClient).(*http.Client)
	return cli.Do(r)
}

// ModifyResponse neutralises any proxied response that a browser might render
// as HTML.  The cluster's Prometheus is attacker-controlled: a customer with
// cluster-admin on their own cluster can make Prometheus return
// `Content-Type: text/html` carrying inline <script> or event-handler
// attributes.  Because the portal serves this proxy on its own origin, any such
// markup would otherwise execute with the authenticated engineer's authority.
//
// The legitimate proxied endpoints (/-/ready and /prometheus/api/...) only ever
// return text/plain or JSON; the Prometheus web UI itself is served from the
// portal's own embedded assets, not through this proxy.  We therefore never
// need to render HTML here.  For any response whose media type is text/html (or
// is missing/sniffable), we force a non-renderable content type and attach a
// restrictive Content-Security-Policy so the engineer's browser cannot execute
// cluster-authored script.
func (p *Prometheus) ModifyResponse(r *http.Response) error {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType == "" || mediaType == "text/html" {
		r.Header.Set("Content-Type", "text/plain; charset=utf-8")
	}

	// Defence in depth: prevent MIME-sniffing into HTML and forbid the browser
	// from loading or executing any resource (including inline script) should
	// this body ever reach a rendering context.
	r.Header.Set("X-Content-Type-Options", "nosniff")
	r.Header.Set("Content-Security-Policy", "default-src 'none'; sandbox")

	return nil
}

func (p *Prometheus) error(r *http.Request, statusCode int, err error) {
	if err != nil {
		p.log.Print(utillog.Sanitize(err.Error()))
	}

	w := responsewriter.New(r)
	http.Error(w, http.StatusText(statusCode), statusCode)

	*r = *r.WithContext(context.WithValue(r.Context(), contextKeyResponse, w.Response()))
}
