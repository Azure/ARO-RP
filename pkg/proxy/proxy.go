package proxy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type Server struct {
	Log *logrus.Entry

	CertFile       string
	KeyFile        string
	ClientCertFile string
	Subnet         string
}

func (s *Server) Run() error {
	_, subnet, err := net.ParseCIDR(s.Subnet)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(s.ClientCertFile)
	if err != nil {
		return err
	}

	clientCert, err := x509.ParseCertificate(b)
	if err != nil {
		return err
	}

	pool := x509.NewCertPool()
	pool.AddCert(clientCert)

	cert, err := ioutil.ReadFile(s.CertFile)
	if err != nil {
		return err
	}

	b, err = ioutil.ReadFile(s.KeyFile)
	if err != nil {
		return err
	}

	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return err
	}

	l, err := tls.Listen("tcp", ":8443", &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					cert,
				},
				PrivateKey: key,
			},
		},
		ClientCAs:  pool,
		ClientAuth: tls.RequireAndVerifyClientCert,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true,
		MinVersion:               tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	})
	if err != nil {
		return err
	}

	return http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		ip, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !subnet.Contains(net.ParseIP(ip)) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		proxy(s.Log, w, r)
	}))
}

func proxy(log *logrus.Entry, w http.ResponseWriter, r *http.Request) {
	c2, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer c2.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	c1, buf, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer c1.Close()
	ch := make(chan struct{})

	go func() {
		defer recover.Panic(log)
		defer close(ch)
		defer func() {
			_ = c2.(*net.TCPConn).CloseWrite()
		}()
		_, _ = io.Copy(c2, buf)
	}()

	func() {
		defer func() {
			_ = c1.(*tls.Conn).CloseWrite()
		}()
		_, _ = io.Copy(c1, c2)
	}()

	<-ch
}
