package prometheus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/portal/util/clientcache"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
)

type prometheus struct {
	log *logrus.Entry

	dbOpenShiftClusters database.OpenShiftClusters

	dialer      proxy.Dialer
	clientCache clientcache.ClientCache
}

func New(baseLog *logrus.Entry,
	dbOpenShiftClusters database.OpenShiftClusters,
	dialer proxy.Dialer,
	aadAuthenticatedRouter *mux.Router) *prometheus {
	p := &prometheus{
		log: baseLog,

		dbOpenShiftClusters: dbOpenShiftClusters,

		dialer:      dialer,
		clientCache: clientcache.New(time.Hour),
	}

	rp := &httputil.ReverseProxy{
		Director:       p.director,
		Transport:      roundtripper.RoundTripperFunc(p.roundTripper),
		ModifyResponse: p.modifyResponse,
		ErrorLog:       log.New(p.log.Writer(), "", 0),
	}

	aadAuthenticatedRouter.NewRoute().Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/prometheus").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path += "/"
		http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
	})

	aadAuthenticatedRouter.NewRoute().PathPrefix("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/prometheus/").Handler(rp)

	return p
}
