package proxy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type conn struct {
	net.Conn
	r *bufio.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

type prod struct{}

func (p *prod) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext(ctx, network, address)
}

type dev struct {
	proxyPool       *x509.CertPool
	proxyClientCert []byte
	proxyClientKey  *rsa.PrivateKey
}

func (d *dev) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("unimplemented network %q", network)
	}

	c, err := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext(ctx, network, os.Getenv("PROXY_HOSTNAME")+":443")
	if err != nil {
		return nil, err
	}

	c = tls.Client(c, &tls.Config{
		RootCAs: d.proxyPool,
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					d.proxyClientCert,
				},
				PrivateKey: d.proxyClientKey,
			},
		},
		ServerName: "proxy",
	})

	err = c.(*tls.Conn).Handshake()
	if err != nil {
		c.Close()
		return nil, err
	}

	r := bufio.NewReader(c)

	req, err := http.NewRequest(http.MethodConnect, "", nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	req.Host = address

	err = req.Write(c)
	if err != nil {
		c.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(r, req)
	if err != nil {
		c.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		c.Close()
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return &conn{Conn: c, r: r}, nil
}

// NewDialer returns a Dialer which can dial a customer cluster API server. When
// not in local development mode, there is no magic here.  In local development
// mode, this dials the development proxy, which proxies to the requested
// endpoint.  This enables the RP to run without routeability to its vnet in
// local development mode.
func NewDialer(isLocalDevelopmentMode bool) (Dialer, error) {
	if !isLocalDevelopmentMode {
		return &prod{}, nil
	}

	d := &dev{}

	basepath := os.Getenv("ARO_CHECKOUT_PATH")
	if basepath == "" {
		// This assumes we are running from an ARO-RP checkout in development
		var err error
		_, curmod, _, _ := runtime.Caller(0)
		basepath, err = filepath.Abs(filepath.Join(filepath.Dir(curmod), "../.."))
		if err != nil {
			return nil, err
		}
	}

	b, err := os.ReadFile(path.Join(basepath, "secrets/proxy.crt"))
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(b)
	if err != nil {
		return nil, err
	}

	d.proxyPool = x509.NewCertPool()
	d.proxyPool.AddCert(cert)

	d.proxyClientCert, err = os.ReadFile(path.Join(basepath, "secrets/proxy-client.crt"))
	if err != nil {
		return nil, err
	}

	b, err = os.ReadFile(path.Join(basepath, "secrets/proxy-client.key"))
	if err != nil {
		return nil, err
	}

	d.proxyClientKey, err = x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}

	return d, nil
}
