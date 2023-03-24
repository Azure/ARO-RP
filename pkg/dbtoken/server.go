package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

var rxValidPermission = regexp.MustCompile("^[a-z]{1,20}$")

type Server interface {
	Run(context.Context) error
}

type server struct {
	env                     env.Core
	log                     *logrus.Entry
	accessLog               *logrus.Entry
	l                       net.Listener
	verifier                oidc.Verifier
	permissionClientFactory func(userid string) cosmosdb.PermissionClient
	m                       metrics.Emitter
}

func NewServer(
	ctx context.Context,
	env env.Core,
	log *logrus.Entry,
	accessLog *logrus.Entry,
	l net.Listener,
	servingKey *rsa.PrivateKey,
	servingCerts []*x509.Certificate,
	verifier oidc.Verifier,
	userc cosmosdb.UserClient,
	m metrics.Emitter,
) (Server, error) {
	config := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey: servingKey,
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

	for _, cert := range servingCerts {
		config.Certificates[0].Certificate = append(config.Certificates[0].Certificate, cert.Raw)
	}

	return &server{
		env:       env,
		log:       log,
		accessLog: accessLog,
		l:         tls.NewListener(l, config),
		verifier:  verifier,
		permissionClientFactory: func(userid string) cosmosdb.PermissionClient {
			return cosmosdb.NewPermissionClient(userc, userid)
		},
		m: m,
	}, nil
}

var healthProbe http.Handler = http.HandlerFunc(healthCheck)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	// empty 200
}

func (s *server) Run(ctx context.Context) error {
	go heartbeat.EmitHeartbeat(s.log, s.m, "dbtoken.heartbeat", nil, func() bool { return true })

	logHandler := Log(s.accessLog)
	panicMiddleware := middleware.Panic(s.log)

	muxRouter := http.NewServeMux()

	muxRouter.Handle("/healthz/ready", panicMiddleware(logHandler(healthProbe)))

	tokenRefresh := panicMiddleware(s.authenticate(logHandler(http.HandlerFunc(s.token))))
	muxRouter.Handle("/token", tokenRefresh)

	srv := &http.Server{
		Handler:     muxRouter,
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute,
		ErrorLog:    log.New(s.log.Writer(), "", 0),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	return srv.Serve(s.l)
}

func (s *server) authenticate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		token, err := s.verifier.Verify(ctx, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		if valid := uuid.IsValid(token.Subject()); !valid {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		ctx = context.WithValue(ctx, middleware.ContextKeyUsername, token.Subject())
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

func (s *server) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	permission := r.URL.Query().Get("permission")
	if !rxValidPermission.MatchString(permission) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	username, _ := ctx.Value(middleware.ContextKeyUsername).(string)
	permc := s.permissionClientFactory(username)

	perm, err := permc.Get(ctx, permission)
	if err != nil {
		s.log.Error(err)
		if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	e := json.NewEncoder(w)
	e.SetIndent("", "    ")

	_ = e.Encode(&tokenResponse{
		Token: perm.Token,
	})
}
