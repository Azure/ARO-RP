package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

type logResponseWriter struct {
	http.ResponseWriter

	statusCode int
	bytes      int
}

func (w *logResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *logResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

type logReadCloser struct {
	io.ReadCloser

	bytes int
}

func (rc *logReadCloser) Read(b []byte) (int, error) {
	n, err := rc.ReadCloser.Read(b)
	rc.bytes += n
	return n, err
}

func Log(baseLog *logrus.Entry) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := time.Now()

			vars := mux.Vars(r)

			r.Body = &logReadCloser{ReadCloser: r.Body}
			w = &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			correlationData := &api.CorrelationData{
				ClientRequestID: r.Header.Get("X-Ms-Client-Request-Id"),
				CorrelationID:   r.Header.Get("X-Ms-Correlation-Request-Id"),
				RequestID:       uuid.NewV4().String(),
				RequestTime:     t,
			}

			if vars["api-version"] == admin.APIVersion ||
				strings.HasPrefix(r.URL.Path, "/admin") {
				correlationData.ClientPrincipalName = r.Header.Get("X-Ms-Client-Principal-Name")
			}

			w.Header().Set("X-Ms-Request-Id", correlationData.RequestID)

			log := baseLog
			log = utillog.EnrichWithPath(log, r.URL.Path)
			log = utillog.EnrichWithCorrelationData(log, correlationData)

			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyLog, log)
			ctx = context.WithValue(ctx, ContextKeyCorrelationData, correlationData)

			r = r.WithContext(ctx)

			defer func() {
				log.WithFields(logrus.Fields{
					"body_read_bytes":      r.Body.(*logReadCloser).bytes,
					"body_written_bytes":   w.(*logResponseWriter).bytes,
					"duration":             time.Now().Sub(t).Seconds(),
					"request_method":       r.Method,
					"request_path":         r.URL.Path,
					"request_proto":        r.Proto,
					"request_remote_addr":  r.RemoteAddr,
					"request_user_agent":   r.UserAgent(),
					"response_status_code": w.(*logResponseWriter).statusCode,
				}).Print("sent response")
			}()

			log.WithFields(logrus.Fields{
				"request_method":      r.Method,
				"request_path":        r.URL.Path,
				"request_proto":       r.Proto,
				"request_remote_addr": r.RemoteAddr,
				"request_user_agent":  r.UserAgent(),
			}).Print("read request")

			h.ServeHTTP(w, r)
		})
	}
}
