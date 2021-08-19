package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/pires/go-proxyproto"

	"github.com/Azure/ARO-RP/pkg/proxy"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

// handleConnect handles incoming HTTP proxy HTTPS CONNECT requests.  The Host
// header will indicate where the incoming connection wants to be connected to.
func (g *gateway) handleConnect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	conn, ok := ctx.Value(contextKeyConnection).(*proxyproto.Conn)
	if !ok {
		g.log.Error("invalid contextKeyConnection")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	clusterResourceID, isAllowed, err := g.isAllowed(conn, host)
	if err != nil {
		g.log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log := utillog.EnrichWithResourceID(g.accessLog, clusterResourceID)
	log = log.WithField("hostname", host)

	if !isAllowed || port != "443" {
		log.Print("access denied")
		g.m.EmitGauge("gateway.connections", 1, map[string]string{
			"protocol": "http",
			"action":   "denied",
		})
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	log.Print("access allowed")
	g.m.EmitGauge("gateway.connections", 1, map[string]string{
		"protocol": "http",
		"action":   "allowed",
	})

	atomic.AddInt64(&g.httpConnections, 1)
	defer atomic.AddInt64(&g.httpConnections, -1)

	proxy.Proxy(g.log, w, r, SocketSize)
}

func (g *gateway) checkReady(w http.ResponseWriter, r *http.Request) {
	if !g.isReady() {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (g *gateway) isReady() bool {
	_, ok := g.lastChangefeed.Load().(time.Time)
	return ok && g.ready.Load().(bool)
}
