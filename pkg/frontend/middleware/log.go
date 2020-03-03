package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
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

			r.Body = &logReadCloser{ReadCloser: r.Body}
			w = &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			correlationID := r.Header.Get("X-Ms-Correlation-Request-Id")

			requestID := uuid.NewV4().String()
			w.Header().Set("X-Ms-Request-Id", requestID)

			log := baseLog.WithFields(logrus.Fields{"correlation_id": correlationID, "request_id": requestID})
			r = r.WithContext(context.WithValue(r.Context(), ContextKeyLog, log))

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
