package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	_ "github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type statusCodeError int

func (err statusCodeError) Error() string {
	return fmt.Sprintf("%d", err)
}

type frontend struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database

	l net.Listener
	s *http.Server

	ready atomic.Value
}

// Runnable represents a runnable object
type Runnable interface {
	Run(<-chan struct{}, chan<- struct{})
}

// NewFrontend returns a new runnable frontend
func NewFrontend(ctx context.Context, baseLog *logrus.Entry, env env.Interface, db *database.Database) (Runnable, error) {
	var err error

	f := &frontend{
		baseLog: baseLog,
		env:     env,
		db:      db,
	}

	l, err := f.env.Listen()
	if err != nil {
		return nil, err
	}

	key, certs, err := f.env.GetSecret(ctx, "rp-server")
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey: key,
			},
		},
		NextProtos: []string{"h2", "http/1.1"},
		ClientAuth: tls.RequestClientCert,
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
	}

	for _, cert := range certs {
		config.Certificates[0].Certificate = append(config.Certificates[0].Certificate, cert.Raw)
	}

	f.l = tls.NewListener(l, config)

	f.ready.Store(true)

	return f, nil
}

func (f *frontend) unauthenticatedRoutes(r *mux.Router) {
	r.Path("/healthz/ready").Methods(http.MethodGet).HandlerFunc(f.getReady)
}

func (f *frontend) authenticatedRoutes(r *mux.Router) {
	s := r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodDelete).HandlerFunc(f.deleteOpenShiftCluster)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftCluster)
	s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchOpenShiftCluster)
	s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchOpenShiftCluster)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/locations/{location}/operations/{operationId}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAsyncOperation)

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/locations/{location}/operationresults/{operationId}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAsyncOperationResult)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/listCredentials").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterCredentials)

	s = r.
		Path("/providers/{resourceProviderNamespace}/operations").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOperations)

	s = r.
		Path("/subscriptions/{subscriptionId}").
		Queries("api-version", "2.0").
		Subrouter()

	s.Methods(http.MethodPut).HandlerFunc(f.putSubscription)
}

func (f *frontend) Run(stop <-chan struct{}, done chan<- struct{}) {
	defer recover.Panic(f.baseLog)

	go func() {
		defer recover.Panic(f.baseLog)

		<-stop

		// mark not ready and wait for ((#probes + 1) * interval + margin) to
		// stop receiving new connections
		f.baseLog.Print("marking not ready and waiting 20 seconds")
		f.ready.Store(false)
		time.Sleep(20 * time.Second)

		// initiate server shutdown and wait for (longest connection timeout +
		// margin) for connections to complete
		f.baseLog.Print("shutting down and waiting up to 65 seconds")
		ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
		defer cancel()

		err := f.s.Shutdown(ctx)
		if err != nil {
			f.baseLog.Error(err)
		}

		close(done)
	}()

	r := mux.NewRouter()
	r.Use(middleware.Log(f.baseLog))
	r.Use(middleware.Panic)
	r.Use(middleware.Headers(f.env))
	r.Use(middleware.Validate(f.env))
	r.Use(middleware.Body)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The requested path could not be found.")
	})
	r.NotFoundHandler = middleware.Authenticated(f.env)(r.NotFoundHandler)

	unauthenticated := r.NewRoute().Subrouter()
	f.unauthenticatedRoutes(unauthenticated)

	authenticated := r.NewRoute().Subrouter()
	authenticated.Use(middleware.Authenticated(f.env))
	f.authenticatedRoutes(authenticated)

	f.s = &http.Server{
		Handler:      middleware.Lowercase(r),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: time.Minute,
		IdleTimeout:  2 * time.Minute,
		ErrorLog:     log.New(f.baseLog.Writer(), "", 0),
	}

	err := f.s.Serve(f.l)
	if err != http.ErrServerClosed {
		f.baseLog.Error(err)
	}
}

func reply(log *logrus.Entry, w http.ResponseWriter, header http.Header, b []byte, err error) {
	for k, v := range header {
		w.Header()[k] = v
	}

	if err != nil {
		switch err := err.(type) {
		case *api.CloudError:
			api.WriteCloudError(w, err)
		case statusCodeError:
			w.WriteHeader(int(err))
		default:
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		}
		return
	}

	if b != nil {
		w.Write(b)
		w.Write([]byte{'\n'})
	}
}
