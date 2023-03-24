package prometheus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"log"
	"net/http/httputil"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/portal/util/clientcache"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
)

type Prometheus struct {
	log *logrus.Entry

	dbOpenShiftClusters database.OpenShiftClusters

	dialer      proxy.Dialer
	clientCache clientcache.ClientCache

	ReverseProxy *httputil.ReverseProxy
}

func New(baseLog *logrus.Entry,
	dbOpenShiftClusters database.OpenShiftClusters,
	dialer proxy.Dialer,
) *Prometheus {
	p := &Prometheus{
		log: baseLog,

		dbOpenShiftClusters: dbOpenShiftClusters,

		dialer:      dialer,
		clientCache: clientcache.New(time.Hour),
	}
	p.ReverseProxy = &httputil.ReverseProxy{
		Director:       p.Director,
		Transport:      roundtripper.RoundTripperFunc(p.RoundTripper),
		ModifyResponse: p.ModifyResponse,
		ErrorLog:       log.New(p.log.Writer(), "", 0),
	}

	return p
}
