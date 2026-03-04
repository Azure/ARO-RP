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
	"io/fs"
	"log"
	"net"
	"net/http"
	"regexp"
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
	"github.com/Azure/ARO-RP/pkg/portal/assets"
	"github.com/Azure/ARO-RP/pkg/portal/cluster"
	"github.com/Azure/ARO-RP/pkg/portal/kubeconfig"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/prometheus"
	"github.com/Azure/ARO-RP/pkg/portal/ssh"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
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

	groupIDs         []string
	elevatedGroupIDs []string

	dbGroup portalDBs

	dialer proxy.Dialer

	templateV2         *template.Template
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
	elevatedGroupIDs []string,
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

		groupIDs:         groupIDs,
		elevatedGroupIDs: elevatedGroupIDs,

		dbGroup: dbGroup,

		dialer: dialer,

		m: m,
	}
}

func (p *portal) setupRouter(kconfig *kubeconfig.Kubeconfig, prom *prometheus.Prometheus, sshStruct *ssh.SSH) (*mux.Router, error) {
	r := mux.NewRouter()
	r.Use(middleware.Panic(p.log))

	assetv2, err := assets.EmbeddedFiles.ReadFile("v2/build/index.html")
	if err != nil {
		return nil, err
	}

	assetPrometheus, err := assets.EmbeddedFiles.ReadFile("prometheus-ui/index.html")
	if err != nil {
		return nil, err
	}

	p.templateV2, err = template.New("index.html").Parse(string(assetv2))
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

	allGroups := append([]string{}, p.groupIDs...)
	allGroups = append(allGroups, p.elevatedGroupIDs...)

	p.aad, err = middleware.NewAAD(p.log, p.auditLog, p.outelAuditClient, p.env, p.baseAccessLog, p.hostname, p.sessionKey, p.clientID, p.clientKey, p.clientCerts, allGroups, unauthenticatedRouter, p.verifier)
	if err != nil {
		return nil, err
	}

	aadAuthenticatedRouter := r.NewRoute().Subrouter()
	aadAuthenticatedRouter.Use(p.aad.AAD)
	aadAuthenticatedRouter.Use(middleware.Log(p.env, p.auditLog, p.baseAccessLog, p.outelAuditClient))
	aadAuthenticatedRouter.Use(p.aad.CheckAuthentication)
	aadAuthenticatedRouter.Use(csrf.Protect(p.sessionKey, csrf.SameSite(csrf.SameSiteStrictMode), csrf.MaxAge(0), csrf.Path("/")))

	p.aadAuthenticatedRoutes(aadAuthenticatedRouter, prom, kconfig, sshStruct)

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

	ssh, err := ssh.New(p.env, p.log, p.baseAccessLog, p.sshl, p.sshKey, p.elevatedGroupIDs, dbOpenShiftClusters, dbPortal, p.dialer)
	if err != nil {
		return nil, nil, nil, err
	}

	err = ssh.Run()
	if err != nil {
		return nil, nil, nil, err
	}

	k := kubeconfig.New(p.log, p.auditLog, p.outelAuditClient, p.env, p.baseAccessLog, p.servingCerts[0], p.elevatedGroupIDs, dbOpenShiftClusters, dbPortal, p.dialer)

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

func (p *portal) aadAuthenticatedRoutes(r *mux.Router, prom *prometheus.Prometheus, kconfig *kubeconfig.Kubeconfig, sshStruct *ssh.SSH) {
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

	r.Methods(http.MethodGet).Path("/api/clusters").HandlerFunc(p.clusters)
	r.Methods(http.MethodGet).Path("/api/info").HandlerFunc(p.info)
	r.Methods(http.MethodGet).Path("/api/regions").HandlerFunc(p.regions)

	// Cluster-specific routes
	r.Path("/api/{subscription}/{resourceGroup}/{clusterName}/clusteroperators").HandlerFunc(p.clusterOperators)
	r.Methods(http.MethodGet).Path("/api/{subscription}/{resourceGroup}/{clusterName}").HandlerFunc(p.clusterInfo)
	r.Path("/api/{subscription}/{resourceGroup}/{clusterName}/nodes").HandlerFunc(p.nodes)
	r.Path("/api/{subscription}/{resourceGroup}/{clusterName}/machines").HandlerFunc(p.machines)
	r.Path("/api/{subscription}/{resourceGroup}/{clusterName}/machine-sets").HandlerFunc(p.machineSets)
	r.Path("/api/{subscription}/{resourceGroup}/{clusterName}/statistics/{statisticsType}").HandlerFunc(p.statistics)
	r.Path("/api/{subscription}/{resourceGroup}/{clusterName}").HandlerFunc(p.clusterInfo)

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

	// kubeconfig
	if kconfig != nil {
		r.Methods(http.MethodPost).Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/kubeconfig/new").HandlerFunc(kconfig.New)
	}

	// ssh
	r.Methods(http.MethodPost).Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/ssh/new").HandlerFunc(sshStruct.New)

	for _, name := range names {
		regexp, _ := regexp.Compile(`v2/build/.*\..*`)
		name := regexp.FindString(name)
		switch name {
		case "v2/build/index.html":
			r.Methods(http.MethodGet).Path("/").HandlerFunc(p.indexV2)
			r.Methods(http.MethodGet).PathPrefix("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}").HandlerFunc(p.indexV2)
		case "":
		default:
			fmtName := strings.TrimPrefix(name, "v2/build/")
			r.Methods(http.MethodGet).Path("/" + fmtName).HandlerFunc(p.serve(name))
		}
	}
}

func (p *portal) indexV2(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}

	err := p.templateV2.ExecuteTemplate(buf, "index.html", map[string]interface{}{
		"location":       p.env.Location(),
		csrf.TemplateTag: csrf.TemplateField(r),
	})
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(buf.Bytes()))
}

func (p *portal) indexPrometheus(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}

	err := p.templatePrometheus.ExecuteTemplate(buf, "index.html", map[string]interface{}{
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
	dbOpenShiftClusters, err := p.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, err
	}

	apiVars := mux.Vars(r)
	subscriptionID := apiVars["subscription"]
	resourceGroup := apiVars["resourceGroup"]
	clusterName := apiVars["clusterName"]
	resourceID := p.getResourceID(subscriptionID, resourceGroup, clusterName)
	if !validate.RxClusterID.MatchString(resourceID) {
		return nil, fmt.Errorf("invalid resource ID")
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	// In development mode, we can have localhost "fake" APIServers which don't
	// get proxied, so use a direct dialer for this
	var dialer proxy.Dialer
	if p.env.IsLocalDevelopmentMode() && doc.OpenShiftCluster.Properties.APIServerProfile.IP == "127.0.0.1" {
		dialer, err = proxy.NewDialer(false, p.log)
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
		asset, err := assets.EmbeddedFiles.ReadFile(path)
		if err != nil {
			p.internalServerError(w, err)
			return
		}

		http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(asset))
	}
}

func (p *portal) getResourceID(subscriptionID, resourceGroup, clusterName string) string {
	return strings.ToLower(
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s",
			subscriptionID, resourceGroup, clusterName))
}

func (p *portal) internalServerError(w http.ResponseWriter, err error) {
	p.log.Warn(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (p *portal) badRequest(w http.ResponseWriter, err error) {
	p.log.Debug(err)
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}
