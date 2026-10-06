package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	frontendmiddleware "github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/portal/assets"
	"github.com/Azure/ARO-RP/pkg/portal/kubeconfig"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/prometheus"
	"github.com/Azure/ARO-RP/pkg/portal/ssh"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
)

type portalDBs interface {
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithPortal
}

type Runnable interface {
	Run(context.Context) error
}

type portal struct {
	env              env.Core
	auditLog         *logrus.Entry
	log              *logrus.Entry
	baseAccessLog    *logrus.Entry
	outelAuditClient audit.Client
	l                net.Listener
	sshl             net.Listener
	verifier         oidc.Verifier

	hostname     string
	servingKey   *rsa.PrivateKey
	servingCerts []*x509.Certificate
	clientID     string
	clientKey    *rsa.PrivateKey
	clientCerts  []*x509.Certificate
	sessionKey   []byte
	sshKey       *rsa.PrivateKey

	groupIDs []string

	dbGroup portalDBs

	dialer proxy.Dialer

	templatePrometheus *template.Template

	aad middleware.AAD

	m metrics.Emitter
}

func NewPortal(env env.Core,
	auditLog *logrus.Entry,
	log *logrus.Entry,
	baseAccessLog *logrus.Entry,
	outelAuditClient audit.Client,
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
	dbGroup portalDBs,
	dialer proxy.Dialer,
	m metrics.Emitter,
) Runnable {
	return &portal{
		env:              env,
		auditLog:         auditLog,
		log:              log,
		baseAccessLog:    baseAccessLog,
		outelAuditClient: outelAuditClient,
		l:                l,
		sshl:             sshl,
		verifier:         verifier,

		hostname:     hostname,
		servingKey:   servingKey,
		servingCerts: servingCerts,
		clientID:     clientID,
		clientKey:    clientKey,
		clientCerts:  clientCerts,
		sessionKey:   sessionKey,
		sshKey:       sshKey,

		groupIDs: groupIDs,

		dbGroup: dbGroup,

		dialer: dialer,

		m: m,
	}
}

func (p *portal) setupRouter(kconfig *kubeconfig.Kubeconfig, prom *prometheus.Prometheus, sshStruct *ssh.SSH) (*mux.Router, error) {
	r := mux.NewRouter()
	r.Use(middleware.Panic(p.log))
	r.Use(middleware.SecurityHeaders())

	assetPrometheus, err := assets.EmbeddedFiles.ReadFile("prometheus-ui/index.html")
	if err != nil {
		return nil, err
	}

	p.templatePrometheus, err = template.New("index.html").Parse(string(assetPrometheus))
	if err != nil {
		return nil, err
	}

	unauthenticatedRouter := r.NewRoute().Subrouter()
	bearerRoutes(unauthenticatedRouter, kconfig)
	p.unauthenticatedRoutes(unauthenticatedRouter)

	p.aad, err = middleware.NewAAD(p.log, p.auditLog, p.outelAuditClient, p.env, p.baseAccessLog, p.hostname, p.sessionKey, p.clientID, p.clientKey, p.clientCerts, p.groupIDs, unauthenticatedRouter, p.verifier)
	if err != nil {
		return nil, err
	}

	aadAuthenticatedRouter := r.NewRoute().Subrouter()
	aadAuthenticatedRouter.Use(p.aad.AAD)
	aadAuthenticatedRouter.Use(middleware.Log(p.env, p.auditLog, p.baseAccessLog, p.outelAuditClient))
	aadAuthenticatedRouter.Use(p.aad.CheckAuthentication)
	aadAuthenticatedRouter.Use(csrf.Protect(p.sessionKey, csrf.SameSite(csrf.SameSiteStrictMode), csrf.MaxAge(0), csrf.Path("/")))

	p.aadAuthenticatedRoutes(aadAuthenticatedRouter, prom)

	return r, nil
}

func (p *portal) setupServices() (*kubeconfig.Kubeconfig, *prometheus.Prometheus, *ssh.SSH, error) {
	dbOpenShiftClusters, err := p.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, nil, nil, err
	}

	dbPortal, err := p.dbGroup.Portal()
	if err != nil {
		return nil, nil, nil, err
	}

	ssh, err := ssh.New(p.env, p.log, p.baseAccessLog, p.sshl, p.sshKey, dbOpenShiftClusters, dbPortal, p.dialer)
	if err != nil {
		return nil, nil, nil, err
	}

	err = ssh.Run()
	if err != nil {
		return nil, nil, nil, err
	}

	k := kubeconfig.New(p.log, p.auditLog, p.outelAuditClient, p.env, p.baseAccessLog, p.servingCerts[0], dbOpenShiftClusters, dbPortal, p.dialer)

	prom := prometheus.New(p.log, dbOpenShiftClusters, p.dialer)

	return k, prom, ssh, nil
}

func (p *portal) Run(ctx context.Context) error {
	config := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey: p.servingKey,
			},
		},
		NextProtos:             []string{"h2", "http/1.1"},
		SessionTicketsDisabled: true,
		MinVersion:             tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}

	for _, cert := range p.servingCerts {
		config.Certificates[0].Certificate = append(config.Certificates[0].Certificate, cert.Raw)
	}

	k, prom, sshStruct, err := p.setupServices()
	if err != nil {
		return err
	}
	router, err := p.setupRouter(k, prom, sshStruct)
	if err != nil {
		return err
	}

	s := &http.Server{
		Handler:     frontendmiddleware.Lowercase(router),
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute,
		ErrorLog:    log.New(p.log.Writer(), "", 0),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	go heartbeat.EmitHeartbeat(p.log, p.m, "portal.heartbeat", nil, func() bool { return true })

	return s.Serve(tls.NewListener(p.l, config))
}

func bearerRoutes(r *mux.Router, k *kubeconfig.Kubeconfig) {
	if k != nil {
		bearerAuthenticatedRouter := r.NewRoute().Subrouter()
		bearerAuthenticatedRouter.Use(middleware.Bearer(k.DbPortal))
		bearerAuthenticatedRouter.Use(middleware.Log(k.Env, k.AuditLog, k.BaseAccessLog, k.OtelAuditClient))

		bearerAuthenticatedRouter.PathPrefix("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/kubeconfig/proxy/").Handler(k.ReverseProxy)
	}
}

func (p *portal) unauthenticatedRoutes(r *mux.Router) {
	logger := middleware.Log(p.env, p.auditLog, p.baseAccessLog, p.outelAuditClient)

	r.Methods(http.MethodGet).Path("/healthz/ready").Handler(logger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
}

func (p *portal) aadAuthenticatedRoutes(r *mux.Router, prom *prometheus.Prometheus) {
	var names []string
	var promNames []string

	err := fs.WalkDir(assets.EmbeddedFiles, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			if strings.HasPrefix(path, "prometheus-ui") {
				promNames = append(promNames, path)
			} else {
				names = append(names, path)
			}
		}
		return nil
	})
	if err != nil {
		p.log.Fatal(err)
	}

	// prometheus
	if prom != nil {
		r.Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/prometheus/-/ready").Handler(prom.ReverseProxy)
		r.PathPrefix("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/prometheus/api/").Handler(prom.ReverseProxy)

		for _, name := range promNames {
			fmtName := strings.TrimPrefix(name, "prometheus-ui/")
			r.Methods(http.MethodGet).Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/prometheus/" + fmtName).HandlerFunc(p.serve(name))
		}

		r.Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/prometheus").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path += "/"
			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
		})
		r.PathPrefix("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/prometheus/").HandlerFunc(p.indexPrometheus)
	}
}

func (p *portal) indexPrometheus(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}

	err := p.templatePrometheus.ExecuteTemplate(buf, "index.html", map[string]interface{}{
		"nonce":          middleware.CSPNonce(r.Context()),
		csrf.TemplateTag: csrf.TemplateField(r),
	})
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(buf.Bytes()))
}

func (p *portal) serve(path string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		asset, err := assets.EmbeddedFiles.ReadFile(path)
		if err != nil {
			p.internalServerError(w, err)
			return
		}

		http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(asset))
	}
}

func (p *portal) internalServerError(w http.ResponseWriter, err error) {
	p.log.Warn(utillog.Sanitize(err.Error()))
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
