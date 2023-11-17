package poc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type FrontendConfig struct {
	Port string
	// TODO(jonachang) delete this in production
	EnableMISE bool
}

type frontend struct {
	logger *logrus.Entry
	port   string
	router chi.Router
}

func NewFrontend(logger *logrus.Entry, config FrontendConfig) frontend {
	var router chi.Router
	if config.EnableMISE {
		router = getMiseRouter()
	} else {
		router = getNonMiseRouter()
	}

	return frontend{
		logger: logger,
		port:   config.Port,
		router: router,
	}
}

func (f *frontend) Run(ctx context.Context) error {
	router := f.router
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

func getBaseRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	return r
}

func getMiseRouter() chi.Router {
	r := getBaseRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		miseToken := extractAuthBearerToken(r.Header)
		miseRespCode, miseRespBody, err := authenticateWithMISE(r.Context(), miseToken, r.Method)
		if err != nil {
			err = fmt.Errorf("unable to perform authentication with MISE: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if miseRespCode != http.StatusOK {
			err = fmt.Errorf("MISE authentication failed with code %d and body %s", miseRespCode, miseRespBody)
			http.Error(w, err.Error(), miseRespCode)
			return
		}
		w.Write([]byte("****** Welcome to ARO-RP on AKS PoC mise ******"))
	})
	return r
}

func getNonMiseRouter() chi.Router {
	r := getBaseRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("****** Welcome to ARO-RP on AKS PoC no mise ******"))
	})
	return r
}

func extractAuthBearerToken(h http.Header) string {
	auth := h.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	return strings.TrimSpace(token)
}
