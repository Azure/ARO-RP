package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync/atomic"

	"github.com/pires/go-proxyproto"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

func (g *gateway) handleHTTPS(ctx context.Context, _c net.Conn) {
	defer recover.Panic(g.log)
	defer _c.Close()

	// 1. Sniff the incoming connection SNI to determine where it wants to go.
	// We return syntheticError to abandon the TLS handshake as soon as we've
	// discovered the server name
	syntheticError := errors.New("synthetic error")

	// wrap the connection with a recorder so we can replay any bytes read by
	// Handshake().
	c1 := newRecorder(_c)
	var serverName string
	err := tls.Server(c1, &tls.Config{
		GetConfigForClient: func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
			serverName = chi.ServerName
			// return syntheticError so that we abandon any further handshaking
			// but can tell that we successfully read the server name.  Note
			// that returning an error here causes pkg/tls to write an alert
			// message back to the client.  This is dropped on the floor by the
			// recorder.
			return nil, syntheticError
		},
	}).Handshake()
	if err != syntheticError {
		// whatever this connection is, it isn't TLS: drop it.  Not much else
		// can be done.
		g.log.Warn(err)
		return
	}

	c1.record = false

	conn, ok := _c.(*proxyproto.Conn)
	if !ok {
		g.log.Error("invalid conn")
		return
	}

	// 2. Determine if we allow the connection.
	clusterResourceID, isAllowed, err := g.isAllowed(conn, serverName)
	if err != nil {
		g.log.Error(err)
		return
	}

	log := utillog.EnrichWithResourceID(g.accessLog, clusterResourceID)
	log = log.WithField("hostname", serverName)

	if !isAllowed {
		log.Print("access denied")
		g.m.EmitGauge("gateway.connections", 1, map[string]string{
			"protocol": "https",
			"action":   "denied",
		})
		return
	}

	log.Print("access allowed")
	g.m.EmitGauge("gateway.connections", 1, map[string]string{
		"protocol": "https",
		"action":   "allowed",
	})

	atomic.AddInt64(&g.httpsConnections, 1)
	defer atomic.AddInt64(&g.httpsConnections, -1)

	// 3. Dial the second leg of the connection (c2).
	c2, err := utilnet.Dial("tcp", serverName+":443", SocketSize)
	if err != nil {
		return
	}

	defer c2.Close()
	ch := make(chan struct{})

	// 4. Proxy c1<->c2.
	go func() {
		defer recover.Panic(g.log)
		defer close(ch)
		defer func() {
			_ = conn.Raw().(*net.TCPConn).CloseWrite()
		}()

		_, _ = io.Copy(c1, c2)
	}()

	func() {
		defer func() {
			_ = c2.(*net.TCPConn).CloseWrite()
		}()

		_, _ = io.Copy(c2, c1)
	}()

	<-ch
}
