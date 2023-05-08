package kubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/util/clientcache"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
)

const (
	kubeconfigNewTimeout = 6 * time.Hour
)

type Kubeconfig struct {
	Log           *logrus.Entry
	BaseAccessLog *logrus.Entry
	Audit         *logrus.Entry

	servingCert      *x509.Certificate
	elevatedGroupIDs []string

	dbOpenShiftClusters database.OpenShiftClusters
	DbPortal            database.Portal

	dialer      proxy.Dialer
	clientCache clientcache.ClientCache
	Env         env.Core

	ReverseProxy *httputil.ReverseProxy
}

func New(baseLog *logrus.Entry,
	audit *logrus.Entry,
	env env.Core,
	baseAccessLog *logrus.Entry,
	servingCert *x509.Certificate,
	elevatedGroupIDs []string,
	dbOpenShiftClusters database.OpenShiftClusters,
	dbPortal database.Portal,
	dialer proxy.Dialer,
) *Kubeconfig {
	k := &Kubeconfig{
		Log:           baseLog,
		BaseAccessLog: baseAccessLog,
		Audit:         audit,

		servingCert:      servingCert,
		elevatedGroupIDs: elevatedGroupIDs,

		dbOpenShiftClusters: dbOpenShiftClusters,
		DbPortal:            dbPortal,

		dialer:      dialer,
		clientCache: clientcache.New(time.Hour),
		Env:         env,
	}

	k.ReverseProxy = &httputil.ReverseProxy{
		Director:  k.director,
		Transport: roundtripper.RoundTripperFunc(k.roundTripper),
		ErrorLog:  log.New(k.Log.Writer(), "", 0),
	}

	return k
}

// New creates a New PortalDocument allowing kubeconfig access to a cluster for
// 6 hours and returns a kubeconfig with the temporary credentials
func (k *Kubeconfig) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resourceID := strings.Join(strings.Split(r.URL.Path, "/")[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) {
		http.Error(w, fmt.Sprintf("invalid resourceId %q", resourceID), http.StatusBadRequest)
		return
	}

	elevated := len(middleware.GroupsIntersect(k.elevatedGroupIDs, ctx.Value(middleware.ContextKeyGroups).([]string))) > 0

	token := k.DbPortal.NewUUID()
	portalDoc := &api.PortalDocument{
		ID:  token,
		TTL: int(kubeconfigNewTimeout / time.Second),
		Portal: &api.Portal{
			Username: ctx.Value(middleware.ContextKeyUsername).(string),
			ID:       resourceID,
			Kubeconfig: &api.Kubeconfig{
				Elevated: elevated,
			},
		},
	}

	_, err := k.DbPortal.Create(ctx, portalDoc)
	if err != nil {
		k.internalServerError(w, err)
		return
	}

	b, err := k.makeKubeconfig("https://"+r.Host+resourceID+"/kubeconfig/proxy", token)
	if err != nil {
		k.internalServerError(w, err)
		return
	}

	filename := strings.Split(r.URL.Path, "/")[8]
	if elevated {
		filename += "-elevated"
	}

	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Disposition", `attachment; filename="`+filename+`.kubeconfig"`)
	_, _ = w.Write(b)
}

func (k *Kubeconfig) internalServerError(w http.ResponseWriter, err error) {
	k.Log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (k *Kubeconfig) makeKubeconfig(server, token string) ([]byte, error) {
	return json.MarshalIndent(&clientcmdv1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: "cluster",
				Cluster: clientcmdv1.Cluster{
					Server: server,
					CertificateAuthorityData: pem.EncodeToMemory(&pem.Block{
						Type:  "CERTIFICATE",
						Bytes: k.servingCert.Raw,
					}),
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: "user",
				AuthInfo: clientcmdv1.AuthInfo{
					Token: token,
				},
			},
		},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: "context",
				Context: clientcmdv1.Context{
					Cluster:   "cluster",
					Namespace: "default",
					AuthInfo:  "user",
				},
			},
		},
		CurrentContext: "context",
	}, "", "    ")
}
