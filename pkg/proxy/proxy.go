package proxy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type Server struct {
	Log *logrus.Entry

	CertFile       string
	KeyFile        string
	ClientCertFile string
	Subnet         string
	subnet         *net.IPNet
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Running.")
}

func (s *Server) Run() error {
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/", health)
	go http.ListenAndServe(":8080", healthMux)

	_, subnet, err := net.ParseCIDR(s.Subnet)
	if err != nil {
		return err
	}
	s.subnet = subnet

	b, err := os.ReadFile(s.ClientCertFile)
	if err != nil {
		return err
	}

	clientCert, err := x509.ParseCertificate(b)
	if err != nil {
		return err
	}

	pool := x509.NewCertPool()
	pool.AddCert(clientCert)

	cert, err := os.ReadFile(s.CertFile)
	if err != nil {
		return err
	}

	b, err = os.ReadFile(s.KeyFile)
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
		ClientCAs:              pool,
		ClientAuth:             tls.RequireAndVerifyClientCert,
		SessionTicketsDisabled: true,
		MinVersion:             tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	})
	if err != nil {
		return err
	}

	return http.Serve(l, http.HandlerFunc(s.proxyHandler))
}

func (s Server) proxyHandler(w http.ResponseWriter, r *http.Request) {
	err := s.validateProxyRequest(w, r)
	if err != nil {
		return
	}
	Proxy(s.Log, w, r, 0)
}

// validateProxyRequest checks that the request is valid. If not, it writes the
// appropriate http headers and returns an error.
func (s Server) validateProxyRequest(w http.ResponseWriter, r *http.Request) error {
	ip, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if r.Method != http.MethodConnect {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return errors.New("request is not valid, method is not CONNECT")
	}

	if !s.subnet.Contains(net.ParseIP(ip)) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return errors.New("request is not allowed, the originating IP is not part of the allowed subnet")
	}

	return nil
}

// Proxy takes an HTTP/1.x CONNECT Request and ResponseWriter from the Golang
// HTTP stack and uses Hijack() to get the underlying Connection (c1).  It dials
// a second Connection (c2) to the requested end Host and then copies data in
// both directions (c1->c2 and c2->c1).
func Proxy(log *logrus.Entry, w http.ResponseWriter, r *http.Request, sz int) {
	c2, err := utilnet.Dial("tcp", r.Host, sz)
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

	// Do as much setup as possible before calling Hijack(), because after
	// Hijack() is called we have no mechanism to report errors back to the
	// caller.

	w.WriteHeader(http.StatusOK)

	c1, buf, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer c1.Close()
	var wg sync.WaitGroup

	// Wait for the c1->c2 goroutine to complete before exiting.
	//Then the deferred c1.Close() and c2.Close() will be called.
	defer wg.Wait()

	wg.Add(1)
	go func() {
		// use a goroutine to copy from c1->c2.  Call c2.CloseWrite() when done.
		defer recover.Panic(log)
		defer wg.Done()
		defer func() {
			conn2, ok := c2.(*net.TCPConn)
			if ok {
				conn2.CloseWrite()
			}
		}()
		_, _ = io.Copy(c2, buf)
	}()

	// copy from c2->c1.  Call c1.CloseWrite() when done.
	defer func() {
		closeWriter, ok := c1.(interface{ CloseWrite() error })
		if ok {
			closeWriter.CloseWrite()
		}
	}()
	_, _ = io.Copy(c1, c2)
}
