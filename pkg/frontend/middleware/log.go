package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
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
			clientRequestID := r.Header.Get("X-Ms-Client-Request-Id")

			requestID := uuid.NewV4().String()
			w.Header().Set("X-Ms-Request-Id", requestID)

			fields := logrus.Fields{
				"correlation_id":    correlationID,
				"request_id":        requestID,
				"client_request_id": clientRequestID,
			}

			updateFieldsFromPath(r.URL.Path, fields)

			log := baseLog.WithFields(fields)
			r = r.WithContext(context.WithValue(r.Context(), ContextKeyLog, log))

			r = r.WithContext(context.WithValue(r.Context(), ContextKeyCorrelationData, api.CorrelationData{
				ClientRequestID: clientRequestID,
				CorrelationID:   correlationID,
				RequestID:       requestID,
			}))

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

var rxTolerantResourceID = regexp.MustCompile(`(?i)^/subscriptions/([^/]+)(?:/resourceGroups/([^/]+)(?:/providers/([^/]+)/([^/]+)(?:/([^/]+))?)?)?`)

func updateFieldsFromPath(path string, fields logrus.Fields) {
	m := rxTolerantResourceID.FindStringSubmatch(path)
	if m == nil {
		return
	}
	if m[1] != "" {
		fields["subscription_id"] = m[1]
	}
	if m[2] != "" {
		fields["resource_group"] = m[2]
	}
	if m[5] != "" {
		fields["resource_name"] = m[5]
		fields["resource_id"] = "/subscriptions/" + m[1] + "/resourceGroups/" + m[2] + "/providers/" + m[3] + "/" + m[4] + "/" + m[5]
	}
}
