package frontend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

type contextKey int

const (
	contextKeyLog contextKey = iota
)

type statsResponseWriter struct {
	statusCode int
	bytes      int

	http.ResponseWriter
}

func (w *statsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *statsResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

type statsReadCloser struct {
	bytes int

	io.ReadCloser
}

func (rc *statsReadCloser) Read(b []byte) (int, error) {
	n, err := rc.ReadCloser.Read(b)
	rc.bytes += n
	return n, err
}

func (f *frontend) middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		b := &statsReadCloser{ReadCloser: r.Body}
		r.Body = b
		w = &statsResponseWriter{ResponseWriter: w}

		correlationID := r.Header.Get("X-Ms-Correlation-Request-Id")
		requestID := uuid.NewV4().String()
		log := f.baseLog.WithFields(logrus.Fields{"correlation-id": correlationID, "request-id": requestID})
		r = r.WithContext(context.WithValue(r.Context(), contextKeyLog, log))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Ms-Request-Id", requestID)

		defer func() {
			if e := recover(); e != nil {
				log.Errorf("panic: %#v\n%s\n", e, string(debug.Stack()))
				f.error(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			}
			log.WithFields(logrus.Fields{
				"access":             true,
				"bodyReceivedBytes":  b.bytes,
				"bodySentBytes":      w.(*statsResponseWriter).bytes,
				"requestDurationMs":  int(time.Now().Sub(t) / time.Millisecond),
				"requestMethod":      r.Method,
				"requestPath":        r.URL.Path,
				"requestProto":       r.Proto,
				"requestRemoteAddr":  r.RemoteAddr,
				"requestUserAgent":   r.UserAgent(),
				"responseStatusCode": w.(*statsResponseWriter).statusCode,
			}).Print()
		}()

		if !f.isValidRequestPath(w, r) {
			return
		}

		if strings.EqualFold(r.Header.Get("X-Ms-Return-Client-Request-Id"), "true") {
			w.Header().Set("X-Ms-Client-Request-Id", r.Header.Get("X-Ms-Client-Request-Id"))
		}

		h.ServeHTTP(w, r)
	})
}

func (f *frontend) error(w http.ResponseWriter, statusCode int, code, target, message string, a ...interface{}) {
	f.cloudError(w, api.NewCloudError(statusCode, code, target, message, a...))
}

func (f *frontend) cloudError(w http.ResponseWriter, err *api.CloudError) {
	w.WriteHeader(err.StatusCode)
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	e.Encode(err)
}
