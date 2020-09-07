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
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Azure/ARO-RP/pkg/env"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

func NewDialer(_env env.Lite) (Dialer, error) {
	if _env.Type() == env.Dev {
		return newProxyDialer()
	}
	return &directDialer{}, nil
}

type conn struct {
	net.Conn
	r *bufio.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

type proxyDialer struct {
	pool *x509.CertPool
	cert []byte
	key  *rsa.PrivateKey
}

func newProxyDialer() (Dialer, error) {
	for _, key := range []string{
		"PROXY_HOSTNAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset (development mode)", key)
		}
	}

	d := &proxyDialer{}

	// This assumes we are running from an ARO-RP checkout in development
	_, curmod, _, _ := runtime.Caller(0)
	basepath, err := filepath.Abs(filepath.Join(filepath.Dir(curmod), "../.."))
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(path.Join(basepath, "secrets/proxy.crt"))
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(b)
	if err != nil {
		return nil, err
	}

	d.pool = x509.NewCertPool()
	d.pool.AddCert(cert)

	d.cert, err = ioutil.ReadFile(path.Join(basepath, "secrets/proxy-client.crt"))
	if err != nil {
		return nil, err
	}

	b, err = ioutil.ReadFile(path.Join(basepath, "secrets/proxy-client.key"))
	if err != nil {
		return nil, err
	}

	d.key, err = x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *proxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
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
		RootCAs: d.pool,
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					d.cert,
				},
				PrivateKey: d.key,
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
