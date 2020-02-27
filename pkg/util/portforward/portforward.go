package portforward

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// DialContext returns a connection to the specified cluster/namespace/pod/port.
func DialContext(ctx context.Context, env env.Interface, oc *api.OpenShiftCluster, namespace, pod, port string) (net.Conn, error) {
	restconfig, err := restconfig.RestConfig(ctx, env, oc)
	if err != nil {
		return nil, err
	}

	// 1. Connect to the API server via private endpoint (and via proxy
	// in development mode)
	clusterURL, err := url.Parse(restconfig.Host)
	if err != nil {
		return nil, err
	}

	rawConn, err := env.DialContext(ctx, "tcp", oc.Properties.NetworkProfile.PrivateEndpointIP+":"+clusterURL.Port())
	if err != nil {
		return nil, err
	}

	// 2. Negotiate TLS
	tlsConfig, err := rest.TLSConfigFor(restconfig)
	if err != nil {
		rawConn.Close()
		return nil, err
	}
	tlsConfig.ServerName = clusterURL.Hostname()

	tlsConn := tls.Client(rawConn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	// 3. Issue an HTTP request to the portforward subresource
	req, err := http.NewRequest(http.MethodPost, "/api/v1/namespaces/"+namespace+"/pods/"+pod+"/portforward", nil)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}
	req.Header.Add(httpstream.HeaderConnection, httpstream.HeaderUpgrade)
	req.Header.Add(httpstream.HeaderUpgrade, spdy.HeaderSpdy31)

	err = req.Write(tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	// 4. Validate the response
	resp, err := http.ReadResponse(bufio.NewReader(tlsConn), req)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	if resp.StatusCode != http.StatusSwitchingProtocols ||
		resp.Header.Get(httpstream.HeaderConnection) != httpstream.HeaderUpgrade ||
		resp.Header.Get(httpstream.HeaderUpgrade) != spdy.HeaderSpdy31 {
		tlsConn.Close()
		return nil, fmt.Errorf("unexpected http response")
	}

	// 5. Negotiate SPDY
	spdyConn, err := spdy.NewClientConnection(tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	// 6. Connect the error stream, r/o
	errorStream, err := spdyConn.CreateStream(http.Header{
		v1.StreamType:                 []string{v1.StreamTypeError},
		v1.PortHeader:                 []string{port},
		v1.PortForwardRequestIDHeader: []string{"0"},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}
	errorStream.Close() // this actually means CloseWrite()

	// 7. Connect the data stream, r/w
	dataStream, err := spdyConn.CreateStream(http.Header{
		v1.StreamType:                 []string{v1.StreamTypeData},
		v1.PortHeader:                 []string{port},
		v1.PortForwardRequestIDHeader: []string{"0"},
	})
	if err != nil {
		spdyConn.Close()
		return nil, err
	}

	return newStreamConn(spdyConn, dataStream, errorStream), nil
}
