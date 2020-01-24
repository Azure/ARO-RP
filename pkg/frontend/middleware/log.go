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

			log := baseLog.WithFields(logrus.Fields{"correlation-id": correlationID, "request-id": requestID})
			r = r.WithContext(context.WithValue(r.Context(), ContextKeyLog, log))

			defer func() {
				log.WithFields(logrus.Fields{
					"access":             true,
					"bodyReadBytes":      r.Body.(*logReadCloser).bytes,
					"bodyWrittenBytes":   w.(*logResponseWriter).bytes,
					"duration":           time.Now().Sub(t).Seconds(),
					"requestMethod":      r.Method,
					"requestPath":        r.URL.Path,
					"requestProto":       r.Proto,
					"requestRemoteAddr":  r.RemoteAddr,
					"requestUserAgent":   r.UserAgent(),
					"responseStatusCode": w.(*logResponseWriter).statusCode,
				}).Print()
			}()

			log.WithFields(logrus.Fields{
				"access":            true,
				"requestMethod":     r.Method,
				"requestPath":       r.URL.Path,
				"requestProto":      r.Proto,
				"requestRemoteAddr": r.RemoteAddr,
				"requestUserAgent":  r.UserAgent(),
			}).Print()

			h.ServeHTTP(w, r)
		})
	}
}
