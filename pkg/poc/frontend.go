package poc

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type frontend struct {
	logger *logrus.Entry
	port   string
	// TODO(jonachang) delete this in production
	enableMISE bool
}

func NewFrontend(logger *logrus.Entry, port string, enableMISE bool) frontend {
	return frontend{
		logger:     logger,
		port:       port,
		enableMISE: enableMISE,
	}
}

func (f *frontend) Run(ctx context.Context) error {
	router := f.getRouter()
	server := &http.Server{
		Addr:     ":" + f.port,
		Handler:  router,
		ErrorLog: log.New(f.logger.Writer(), "", 0),
	}

	go func() {
		f.logger.Info("Starting http server...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			f.logger.Fatalf("Server listen/serve error: %s", err)
		}
	}()

	<-ctx.Done()

	f.logger.Info("Stopping http server")
	err := server.Shutdown(context.Background())
	if err != nil {
		f.logger.Errorf("Server shutdown error: %s", err)
	}

	return err
}

func (f *frontend) getRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		f.logger.Infof("Received request: %s", time.Now().String())
		// TODO(jonachang): remove this when go production.
		if f.enableMISE == true {
			miseToken := extractAuthBearerToken(r.Header)
			miseError := authenticateWithMISE(r.Context(), miseToken)
			if miseError != nil {
				f.logger.Infof("MISE error: %s", miseError)
				w.Write([]byte("****** Blocked by MISE authorization ******"))
			} else {
				w.Write([]byte("****** Welcome to ARO-RP on AKS PoC mise ******"))
			}
		} else {
			w.Write([]byte("****** Welcome to ARO-RP on AKS PoC no mise ******"))
		}
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	return r
}

func extractAuthBearerToken(h http.Header) string {
	auth := h.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	return strings.TrimSpace(token)
}
