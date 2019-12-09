package frontend

import (
	"context"
	"io"
	"io/ioutil"
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
	contextKeyOriginalPath
	contextKeyBody
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

func (f frontend) authenticated(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil ||
			len(r.TLS.PeerCertificates) == 0 ||
			!f.env.IsAuthorized(r.TLS.PeerCertificates[0].Raw) {
			api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
			return
		}

		h.ServeHTTP(w, r)
	})
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
				api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			}
			log.WithFields(logrus.Fields{
				"access":             true,
				"bodyReceivedBytes":  b.bytes,
				"bodySentBytes":      w.(*statsResponseWriter).bytes,
				"durationMs":         int(time.Now().Sub(t) / time.Millisecond),
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

func lowercase(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), contextKeyOriginalPath, r.URL.Path))
		r.URL.Path = strings.ToLower(r.URL.Path)

		h.ServeHTTP(w, r)
	})
}

func readBody(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	if strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0] != "application/json" {
		return nil, api.NewCloudError(http.StatusUnsupportedMediaType, api.CloudErrorCodeUnsupportedMediaType, "", "The content media type '%s' is not supported. Only 'application/json' is supported.", r.Header.Get("Content-Type"))
	}

	body, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1048576))
	if err != nil {
		return nil, api.NewCloudError(http.StatusUnsupportedMediaType, api.CloudErrorCodeInvalidResource, "", "The resource definition is invalid.")

	}

	return r.WithContext(context.WithValue(r.Context(), contextKeyBody, body)), nil
}

func reply(log *logrus.Entry, w http.ResponseWriter, b []byte, err error) {
	if err != nil {
		switch err := err.(type) {
		case *api.CloudError:
			api.WriteCloudError(w, err)
		case *noContent:
			w.WriteHeader(http.StatusNoContent)
		default:
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		}
		return
	}

	if b != nil {
		w.Write(b)
		w.Write([]byte{'\n'})
	}
}
