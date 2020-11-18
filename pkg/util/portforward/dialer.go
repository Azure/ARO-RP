package portforward

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// dialSpdy connects to the specified path on the API server of oc and
// negotiates SPDY
func dialSpdy(ctx context.Context, dialer proxy.Dialer, oc *api.OpenShiftCluster, path string) (httpstream.Connection, error) {
	restconfig, err := restconfig.RestConfig(dialer, oc)
	if err != nil {
		return nil, err
	}

	// 1. Connect to the API server via private endpoint (and via proxy in
	//    development mode)
	clusterURL, err := url.Parse(restconfig.Host)
	if err != nil {
		return nil, err
	}

	rawConn, err := dialer.DialContext(ctx, "tcp", oc.Properties.NetworkProfile.PrivateEndpointIP+":"+clusterURL.Port())
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

	// 3. Issue an HTTP POST request to the specified path
	req, err := http.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}
	req.Header.Add(httpstream.HeaderConnection, httpstream.HeaderUpgrade)
	req.Header.Add(httpstream.HeaderProtocolVersion, portforward.PortForwardProtocolV1Name)
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

	if resp.StatusCode != http.StatusSwitchingProtocols {
		tlsConn.Close()
		return nil, fmt.Errorf("unexpected http status code %d", resp.StatusCode)
	}

	if resp.Header.Get(httpstream.HeaderConnection) != httpstream.HeaderUpgrade {
		tlsConn.Close()
		return nil, fmt.Errorf("unexpected http header %s: %s", httpstream.HeaderConnection, resp.Header.Get(httpstream.HeaderConnection))
	}

	if resp.Header.Get(httpstream.HeaderUpgrade) != spdy.HeaderSpdy31 {
		tlsConn.Close()
		return nil, fmt.Errorf("unexpected http header %s: %s", httpstream.HeaderUpgrade, resp.Header.Get(httpstream.HeaderUpgrade))
	}

	// 5. Negotiate SPDY
	spdyConn, err := spdy.NewClientConnection(tlsConn)
	if err != nil {
		tlsConn.Close()
		return nil, err
	}

	return spdyConn, nil
}
