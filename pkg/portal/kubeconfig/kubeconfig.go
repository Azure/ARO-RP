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

	"github.com/gorilla/mux"
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

type kubeconfig struct {
	log           *logrus.Entry
	baseAccessLog *logrus.Entry

	servingCert      *x509.Certificate
	elevatedGroupIDs []string

	dbOpenShiftClusters database.OpenShiftClusters
	dbPortal            database.Portal

	dialer      proxy.Dialer
	clientCache clientcache.ClientCache
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
	aadAuthenticatedRouter,
	unauthenticatedRouter *mux.Router) *kubeconfig {
	k := &kubeconfig{
		log:           baseLog,
		baseAccessLog: baseAccessLog,

		servingCert:      servingCert,
		elevatedGroupIDs: elevatedGroupIDs,

		dbOpenShiftClusters: dbOpenShiftClusters,
		dbPortal:            dbPortal,

		dialer:      dialer,
		clientCache: clientcache.New(time.Hour),
	}

	rp := &httputil.ReverseProxy{
		Director:  k.director,
		Transport: roundtripper.RoundTripperFunc(k.roundTripper),
		ErrorLog:  log.New(k.log.Writer(), "", 0),
	}

	aadAuthenticatedRouter.NewRoute().Methods(http.MethodPost).Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/kubeconfig/new").HandlerFunc(k.new)

	bearerAuthenticatedRouter := unauthenticatedRouter.NewRoute().Subrouter()
	bearerAuthenticatedRouter.Use(middleware.Bearer(k.dbPortal))
	bearerAuthenticatedRouter.Use(middleware.Log(env, audit, k.baseAccessLog))

	bearerAuthenticatedRouter.PathPrefix("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/kubeconfig/proxy/").Handler(rp)

	return k
}

// new creates a new PortalDocument allowing kubeconfig access to a cluster for
// 6 hours and returns a kubeconfig with the temporary credentials
func (k *kubeconfig) new(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resourceID := strings.Join(strings.Split(r.URL.Path, "/")[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) {
		http.Error(w, fmt.Sprintf("invalid resourceId %q", resourceID), http.StatusBadRequest)
		return
	}

	elevated := len(middleware.GroupsIntersect(k.elevatedGroupIDs, ctx.Value(middleware.ContextKeyGroups).([]string))) > 0

	token := k.dbPortal.NewUUID()
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

	_, err := k.dbPortal.Create(ctx, portalDoc)
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

func (k *kubeconfig) internalServerError(w http.ResponseWriter, err error) {
	k.log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (k *kubeconfig) makeKubeconfig(server, token string) ([]byte, error) {
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
