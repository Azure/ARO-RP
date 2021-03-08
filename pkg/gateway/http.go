package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net"
	"net/http"
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
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	log.Print("access allowed")
	// TODO: some sort of metrics framework to track connections

	proxy.Proxy(g.log, w, r, SocketSize)
}

func (g *gateway) checkReady(w http.ResponseWriter, r *http.Request) {
	if _, ok := g.lastChangefeed.Load().(time.Time); !ok || !g.ready.Load().(bool) {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
