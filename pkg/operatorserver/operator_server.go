package operator_server

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Run runs a simple web server that replies 200.
func Run(log *logrus.Entry) {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "")
	})
	go func() {
		if err := http.ListenAndServe(":8081", nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("aro-operator webserver failed to start: %s\n", err)
		}
	}()
	log.Info("Webserver is listening on 8081")
}
