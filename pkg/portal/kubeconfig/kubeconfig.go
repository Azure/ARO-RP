package kubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"log"
	"net/http/httputil"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/portal/util/clientcache"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	"github.com/Azure/ARO-RP/pkg/util/roundtripper"
)

type Kubeconfig struct {
	Log             *logrus.Entry
	BaseAccessLog   *logrus.Entry
	AuditLog        *logrus.Entry
	OtelAuditClient audit.Client
	servingCert     *x509.Certificate

	dbOpenShiftClusters database.OpenShiftClusters
	DbPortal            database.Portal

	dialer      proxy.Dialer
	clientCache clientcache.ClientCache
	Env         env.Core

	ReverseProxy *httputil.ReverseProxy
}

func New(baseLog *logrus.Entry,
	auditLog *logrus.Entry,
	otelAuditClient audit.Client,
	env env.Core,
	baseAccessLog *logrus.Entry,
	servingCert *x509.Certificate,
	dbOpenShiftClusters database.OpenShiftClusters,
	dbPortal database.Portal,
	dialer proxy.Dialer,
) *Kubeconfig {
	k := &Kubeconfig{
		Log:             baseLog,
		AuditLog:        auditLog,
		OtelAuditClient: otelAuditClient,
		BaseAccessLog:   baseAccessLog,

		servingCert: servingCert,

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
