package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-chi/chi/v5"
	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
)

type Runnable interface {
	Run(context.Context, chan<- struct{})
}

// gateway proxies TCP connections from clusters to controlled destinations.  It
// has two modes of operations:
//
// 1. TLS proxy (https.go).  Gateway sniffs the SNI field in the TLS ClientHello
// message.  If the connection request is allowed, a second connection is opened
// and bytes are copied between the two connections at the TCP level.
//
// 2. HTTP CONNECT proxy (http.go).  Gateway receives an incoming HTTP/1.1
// CONNECT request, handled using the standard go HTTP stack.  If the connection
// request is allowed, the connection is Hijack()ed, a second connection is
// opened and bytes are copied between the two connections at the TCP level.
//
// Mode 2 is a bit of a hack: it is used to enable the bootstrap VM to download
// its ignition config.  At this stage in the bootstrap process it is too early
// to mess with the bootstrap VM's /etc/hosts; we rely on the ability to set
// HTTPS_PROXY in the ignition config instead.
//
// Important note: regardless of mode, TLS traffic is never decrypted/re-
// encrypted by the gateway.
type gateway struct {
	env       env.Core
	log       *logrus.Entry
	accessLog *logrus.Entry

	ready          atomic.Value
	lastChangefeed atomic.Value //time.Time
	mu             sync.RWMutex
	gateways       map[string]*api.Gateway

	dbGateway database.Gateway

	httpsl       net.Listener
	httpl        net.Listener
	httpHealthl  net.Listener
	server       *http.Server
	healthServer *http.Server

	allowList map[string]struct{}

	m                metrics.Emitter
	httpConnections  int64
	httpsConnections int64
}

type contextKey int

const (
	contextKeyConnection contextKey = iota
)

// we could end up handling a lot of long-lived connections in parallel. Let's
// think of our memory overheads up-front.  Back of the envelope sizing: 8GiB
// kernel memory / (2 * 64KiB buffers) / 2 pairs = 32Ki concurrent connection
// pairs.
const SocketSize = 65536

// TODO: may one day want to limit gateway readiness on # active connections

func NewGateway(ctx context.Context, env env.Core, baseLog, accessLog *logrus.Entry, dbGateway database.Gateway, httpsl, httpl, httpHealthl net.Listener, acrResourceID, gatewayDomains string, m metrics.Emitter) (Runnable, error) {
	var domains []string
	if gatewayDomains != "" {
		domains = strings.Split(gatewayDomains, ",")
	}

	for _, rawurl := range []string{
		env.Environment().ActiveDirectoryEndpoint, // e.g. login.microsoftonline.com
		env.Environment().ResourceManagerEndpoint, // e.g. management.azure.com
	} {
		u, err := url.Parse(rawurl)
		if err != nil {
			return nil, err
		}

		if u.Hostname() == "" {
			return nil, errors.New("missing required domain. Ensure the environment has both ActiveDirectoryEndpoint and ResourceManagerEndpoint")
		}

		domains = append(domains, u.Hostname())
	}

	if acrResourceID != "" {
		acrResource, err := azure.ParseResourceID(acrResourceID)
		if err != nil {
			return nil, err
		}

		domains = append(domains,
			acrResource.ResourceName+"."+env.Environment().ContainerRegistryDNSSuffix,                         // e.g. arosvc.azurecr.io
			acrResource.ResourceName+"."+env.Location()+".data."+env.Environment().ContainerRegistryDNSSuffix, // e.g. arosvc.eastus.data.azurecr.io
		)
	}

	allowList := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		allowList[strings.ToLower(domain)] = struct{}{}
	}

	g := &gateway{
		env:       env,
		log:       baseLog,
		accessLog: accessLog,

		gateways: map[string]*api.Gateway{},

		dbGateway: dbGateway,

		// httpsl and httpl are wrapped with proxyproto.Listener so that we can
		// later pick out the private endpoint ID of the incoming connection via
		// Azure's haproxy protocol support
		// (https://docs.microsoft.com/en-us/azure/private-link/private-link-service-overview#getting-connection-information-using-tcp-proxy-v2).
		httpsl: &proxyproto.Listener{
			Listener: httpsl,
		},
		httpl: &proxyproto.Listener{
			Listener: httpl,
		},
		httpHealthl: &proxyproto.Listener{
			Listener: httpHealthl,
		},
		server: &http.Server{
			ReadTimeout: 10 * time.Second,
			IdleTimeout: 2 * time.Minute,
			ErrorLog:    log.New(baseLog.Writer(), "", 0),
			BaseContext: func(net.Listener) context.Context { return ctx },
			ConnContext: func(ctx context.Context, c net.Conn) context.Context {
				// expose the underlying net.Conn of the HTTP request in flight
				// via the contextKeyConnection key.  This allows us to pick out
				// the private endpoint ID from the context of the HTTP request.
				return context.WithValue(ctx, contextKeyConnection, c)
			},
		},
		healthServer: &http.Server{
			ReadTimeout: 10 * time.Second,
			IdleTimeout: 2 * time.Minute,
			ErrorLog:    log.New(baseLog.Writer(), "", 0),
			BaseContext: func(net.Listener) context.Context { return ctx },
		},

		allowList: allowList,
		m:         m,
	}

	panicMiddleware := middleware.Panic(baseLog)

	chiRouter := chi.NewMux()
	chiRouter.Use(panicMiddleware)

	chiRouter.Get("/healthz/ready", http.HandlerFunc(g.checkReady))
	chiRouter.Connect("/*", http.HandlerFunc(g.handleConnect))

	g.server.Handler = chiRouter
	g.healthServer.Handler = chiRouter

	g.ready.Store(true)

	return g, nil
}

func (g *gateway) Run(ctx context.Context, done chan<- struct{}) {
	go g.changefeed(ctx)

	go g.emitMetrics()
	go heartbeat.EmitHeartbeat(g.log, g.m, "gateway.heartbeat", nil, g.isReady)

	go func() {
		// HTTP proxy connections are handled using the go HTTP stack
		_ = g.server.Serve(g.httpl)
	}()

	go func() {
		// listen for health check
		_ = g.healthServer.Serve(g.httpHealthl)
	}()

	go func() {
		for {
			c, err := g.httpsl.Accept()
			if err != nil {
				g.log.Error(err)
				return
			}

			// HTTPS connections are never decrypted, so they are handled like
			// TCP connections
			go g.handleHTTPS(ctx, c)
		}
	}()

	<-ctx.Done()

	// mark not ready and wait for ((#probes + 1) * interval + margin) to stop
	// receiving new connections
	g.log.Print("marking not ready and waiting 45 seconds")
	g.ready.Store(false)
	time.Sleep(45 * time.Second)

	// TODO: wait some more

	close(done)
}
