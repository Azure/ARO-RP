package poc

import (
	"bytes"
	"context"
	"fmt"
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
}

type MiseRequestData struct {
	MiseURL        string
	OriginalURI    string
	OriginalMethod string
	Token          string
}

func NewFrontend(logger *logrus.Entry, port string) frontend {
	return frontend{
		logger: logger,
		port:   port,
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
		handleMISE(w, r)
		w.Write([]byte("****** ARO-RP on AKS PoC frontend******"))
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	return r
}

func handleMISE(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	t := extractToken(r.Header)
	m := MiseRequestData{
		MiseURL:        "http://0.0.0.0:5000/ValidateRequest",
		OriginalURI:    "https://server/endpoint",
		OriginalMethod: r.Method,
		Token:          t,
	}
	req, err := createMiseHTTPRequest(ctx, m)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	log.Default().Println("Response status: ", resp.Status)

	w.WriteHeader(resp.StatusCode)
	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Fprintln(w, "Authorized")
	default:
		fmt.Fprintln(w, "Unauthorized")
	}

}

func extractToken(h http.Header) string {
	log.Default().Println("header is: ", h)
	auth := h.Get("Authorization")
	log.Default().Println("Authorization header is: ", auth)
	token := strings.TrimPrefix(auth, "Bearer ")
	log.Default().Println("token value is: ", token)
	return strings.TrimSpace(token)
}

func createMiseHTTPRequest(ctx context.Context, data MiseRequestData) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, data.MiseURL, bytes.NewBuffer(nil))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	req.Header.Set("Original-URI", data.OriginalURI)
	req.Header.Set("Original-Method", data.OriginalMethod)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", data.Token))
	return req, nil
}
