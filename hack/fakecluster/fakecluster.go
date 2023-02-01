package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	frontendmiddleware "github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/cluster"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func run(ctx context.Context, l *logrus.Entry) error {
	certFile := flag.String("certFile", "secrets/proxy.crt", "file containing server certificate")
	keyFile := flag.String("keyFile", "secrets/proxy.key", "file containing server key")
	port := flag.Int("port", 6443, "Port to listen on")
	host := flag.String("host", "localhost", "Host to listen on")

	l.Printf("starting, git commit %s", version.GitCommit)

	flag.Parse()

	cert, err := os.ReadFile(*certFile)
	if err != nil {
		panic(err)
	}

	b, err := os.ReadFile(*keyFile)
	if err != nil {
		panic(err)
	}

	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		panic(err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					cert,
				},
				PrivateKey: key,
			},
		},
		NextProtos: []string{"h2", "http/1.1"},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		SessionTicketsDisabled: true,
		MinVersion:             tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}

	r := mux.NewRouter()
	r.Use(middleware.Panic(l))

	r.NewRoute().PathPrefix(
		"/api/config.openshift.io/v1/clusteroperators",
	).HandlerFunc(resp(cluster.MustAsset("clusteroperator.json")))

	r.NewRoute().PathPrefix(
		"/api/v1/nodes",
	).HandlerFunc(resp(cluster.MustAsset("nodes.json")))

	s := &http.Server{
		Handler:     frontendmiddleware.Lowercase(r),
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute,
		ErrorLog:    log.New(l.Writer(), "", 0),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	l.Printf("Listening on %s", fmt.Sprint(*host, ":", *port))
	lis, err := net.Listen("tcp", fmt.Sprint(*host, ":", *port))
	if err != nil {
		return err
	}

	return s.Serve(tls.NewListener(lis, config))
}

func resp(bytes []byte) func(http.ResponseWriter, *http.Request) {
	e := func(w http.ResponseWriter, r *http.Request) {
		var resp interface{}
		err := json.Unmarshal(bytes, &resp)
		if err != nil {
			return
		}

		b, err := json.MarshalIndent(resp, "", "    ")
		if err != nil {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}
	return e
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
