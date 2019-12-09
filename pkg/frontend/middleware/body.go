package middleware

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/jim-minter/rp/pkg/api"
)

func Body(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch, http.MethodPost, http.MethodPut:
			if strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0] != "application/json" {
				api.WriteError(w, http.StatusUnsupportedMediaType, api.CloudErrorCodeUnsupportedMediaType, "", "The content media type '%s' is not supported. Only 'application/json' is supported.", r.Header.Get("Content-Type"))
				return
			}

			body, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1048576))
			if err != nil {
				api.WriteError(w, http.StatusUnsupportedMediaType, api.CloudErrorCodeInvalidResource, "", "The resource definition is invalid.")
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), ContextKeyBody, body))
		}

		h.ServeHTTP(w, r)
	})
}
