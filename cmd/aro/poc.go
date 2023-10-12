package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Azure/ARO-RP/pkg/poc"
	"github.com/sirupsen/logrus"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func rpPoc(ctx context.Context, log *logrus.Entry) error {
	log.Print("********** ARO-RP on AKS PoC Testing**********")
	// Start Mise Authorization
	http.HandleFunc("/", handler)

	ctx, shutdown := context.WithCancel(ctx)
	defer shutdown()
	go handleSigterm(log, shutdown)

	port := flag.Arg(1)
	frontEnd := poc.NewFrontend(log, port)

	return frontEnd.Run(ctx)
}

func handleSigterm(log *logrus.Entry, shutdown context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals

	log.Print("received SIGTERM. Terminating...")

	shutdown()
}

type MiseRequestData struct {
	MiseURL        string
	OriginalURI    string
	OriginalMethod string
	Token          string
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	t := extractToken(r.Header)
	m := MiseRequestData{
		MiseURL:        "http://localhost:5000/ValidateRequest",
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

func extractToken(h http.Header) string {
	auth := h.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	return strings.TrimSpace(token)
}
