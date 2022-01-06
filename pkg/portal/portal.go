package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	frontendmiddleware "github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/portal/cluster"
	"github.com/Azure/ARO-RP/pkg/portal/kubeconfig"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/prometheus"
	"github.com/Azure/ARO-RP/pkg/portal/ssh"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
)

type Runnable interface {
	Run(context.Context) error
}

type portal struct {
	env                 env.Core
	audit               *logrus.Entry
	log                 *logrus.Entry
	baseAccessLog       *logrus.Entry
	l                   net.Listener
	sshl                net.Listener
	verifier            oidc.Verifier
	baseRouter          *mux.Router
	authenticatedRouter *mux.Router
	publicRouter        *mux.Router

	hostname     string
	servingKey   *rsa.PrivateKey
	servingCerts []*x509.Certificate
	clientID     string
	clientKey    *rsa.PrivateKey
	clientCerts  []*x509.Certificate
	sessionKey   []byte
	sshKey       *rsa.PrivateKey

	groupIDs         []string
	elevatedGroupIDs []string

	dbPortal            database.Portal
	dbOpenShiftClusters database.OpenShiftClusters

	dialer proxy.Dialer

	t *template.Template

	aad middleware.AAD

	m metrics.Interface
}

func NewPortal(env env.Core,
	audit *logrus.Entry,
	log *logrus.Entry,
	baseAccessLog *logrus.Entry,
	l net.Listener,
	sshl net.Listener,
	verifier oidc.Verifier,
	hostname string,
	servingKey *rsa.PrivateKey,
	servingCerts []*x509.Certificate,
	clientID string,
	clientKey *rsa.PrivateKey,
	clientCerts []*x509.Certificate,
	sessionKey []byte,
	sshKey *rsa.PrivateKey,
	groupIDs []string,
	elevatedGroupIDs []string,
	dbOpenShiftClusters database.OpenShiftClusters,
	dbPortal database.Portal,
	dialer proxy.Dialer,
	m metrics.Interface,
) Runnable {
	return &portal{
		env:           env,
		audit:         audit,
		log:           log,
		baseAccessLog: baseAccessLog,
		l:             l,
		sshl:          sshl,
		verifier:      verifier,

		hostname:     hostname,
		servingKey:   servingKey,
		servingCerts: servingCerts,
		clientID:     clientID,
		clientKey:    clientKey,
		clientCerts:  clientCerts,
		sessionKey:   sessionKey,
		sshKey:       sshKey,

		groupIDs:         groupIDs,
		elevatedGroupIDs: elevatedGroupIDs,

		dbOpenShiftClusters: dbOpenShiftClusters,
		dbPortal:            dbPortal,

		dialer: dialer,

		m: m,
	}
}

func (p *portal) setupRouter() error {
	if p.baseRouter != nil {
		return fmt.Errorf("can't setup twice")
	}

	r := mux.NewRouter()
	r.Use(middleware.Panic(p.log))

	asset, err := Asset("index.html")
	if err != nil {
		return err
	}

	p.t, err = template.New("index.html").Parse(string(asset))
	if err != nil {
		return err
	}

	unauthenticatedRouter := r.NewRoute().Subrouter()
	p.unauthenticatedRoutes(unauthenticatedRouter)

	allGroups := append([]string{}, p.groupIDs...)
	allGroups = append(allGroups, p.elevatedGroupIDs...)

	p.aad, err = middleware.NewAAD(p.log, p.audit, p.env, p.baseAccessLog, p.hostname, p.sessionKey, p.clientID, p.clientKey, p.clientCerts, allGroups, unauthenticatedRouter, p.verifier)
	if err != nil {
		return err
	}

	aadAuthenticatedRouter := r.NewRoute().Subrouter()
	aadAuthenticatedRouter.Use(p.aad.AAD)
	aadAuthenticatedRouter.Use(middleware.Log(p.env, p.audit, p.baseAccessLog))
	aadAuthenticatedRouter.Use(p.aad.CheckAuthentication)
	aadAuthenticatedRouter.Use(csrf.Protect(p.sessionKey, csrf.SameSite(csrf.SameSiteStrictMode), csrf.MaxAge(0), csrf.Path("/")))

	p.aadAuthenticatedRoutes(aadAuthenticatedRouter)

	p.baseRouter = r
	p.publicRouter = unauthenticatedRouter
	p.authenticatedRouter = aadAuthenticatedRouter

	return nil
}

func (p *portal) setupServices() error {
	ssh, err := ssh.New(p.env, p.log, p.baseAccessLog, p.sshl, p.sshKey, p.elevatedGroupIDs, p.dbOpenShiftClusters, p.dbPortal, p.dialer, p.authenticatedRouter)
	if err != nil {
		return err
	}

	err = ssh.Run()
	if err != nil {
		return err
	}

	kubeconfig.New(p.log, p.audit, p.env, p.baseAccessLog, p.servingCerts[0], p.elevatedGroupIDs, p.dbOpenShiftClusters, p.dbPortal, p.dialer, p.authenticatedRouter, p.publicRouter)

	prometheus.New(p.log, p.dbOpenShiftClusters, p.dialer, p.authenticatedRouter)

	return nil
}

func (p *portal) Run(ctx context.Context) error {
	config := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey: p.servingKey,
			},
		},
		NextProtos: []string{"h2", "http/1.1"},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true,
		MinVersion:               tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}

	for _, cert := range p.servingCerts {
		config.Certificates[0].Certificate = append(config.Certificates[0].Certificate, cert.Raw)
	}

	if p.baseRouter == nil {
		err := p.setupRouter()
		if err != nil {
			return err
		}
		err = p.setupServices()
		if err != nil {
			return err
		}
	}

	s := &http.Server{
		Handler:     frontendmiddleware.Lowercase(p.baseRouter),
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute,
		ErrorLog:    log.New(p.log.Writer(), "", 0),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	go heartbeat.EmitHeartbeat(p.log, p.m, "portal.heartbeat", nil, func() bool { return true })

	return s.Serve(tls.NewListener(p.l, config))
}

func (p *portal) unauthenticatedRoutes(r *mux.Router) {
	logger := middleware.Log(p.env, p.audit, p.baseAccessLog)

	r.NewRoute().Methods(http.MethodGet).Path("/healthz/ready").Handler(logger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
}

func (p *portal) aadAuthenticatedRoutes(r *mux.Router) {
	for _, name := range AssetNames() {
		if name == "index.html" {
			r.NewRoute().Methods(http.MethodGet).Path("/").HandlerFunc(p.index)
			continue
		}

		r.NewRoute().Methods(http.MethodGet).Path("/" + name).HandlerFunc(p.serve(name))
	}

	r.NewRoute().Methods(http.MethodGet).Path("/api/clusters").HandlerFunc(p.clusters)
	r.NewRoute().Methods(http.MethodGet).Path("/api/info").HandlerFunc(p.info)

	// Cluster-specific routes
	r.NewRoute().PathPrefix("/api/{subscription}/{resourceGroup}/{name}/clusteroperators").HandlerFunc(p.clusterOperators)
	r.NewRoute().Methods(http.MethodGet).Path("/api/{subscription}/{resourceGroup}/{name}").HandlerFunc(p.clusterInfo)
}

func (p *portal) index(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}

	err := p.t.ExecuteTemplate(buf, "index.html", map[string]interface{}{
		"location":       p.env.Location(),
		csrf.TemplateTag: csrf.TemplateField(r),
	})
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(buf.Bytes()))
}

// makeFetcher creates a cluster.FetchClient suitable for use by the Portal REST API
func (p *portal) makeFetcher(ctx context.Context, r *http.Request) (cluster.FetchClient, error) {
	resourceID := strings.Join(strings.Split(r.URL.Path, "/")[:9], "/")
	if !validate.RxClusterID.MatchString(resourceID) {
		return nil, fmt.Errorf("invalid resource ID")
	}

	doc, err := p.dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	// In development mode, we can have localhost "fake" APIServers which don't
	// get proxied, so use a direct dialer for this
	var dialer proxy.Dialer
	if p.env.IsLocalDevelopmentMode() && doc.OpenShiftCluster.Properties.APIServerProfile.IP == "127.0.0.1" {
		dialer, err = proxy.NewDialer(false)
		if err != nil {
			return nil, err
		}
	} else {
		dialer = p.dialer
	}

	return cluster.NewFetchClient(p.log, dialer, doc)
}

func (p *portal) serve(path string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := Asset(path)
		if err != nil {
			p.internalServerError(w, err)
			return
		}

		http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(b))
	}
}

func (p *portal) internalServerError(w http.ResponseWriter, err error) {
	p.log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
