package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestBody(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		method         string
		header         http.Header
		wantStatusCode int
	}{
		{
			name:           "Get request - No checking",
			method:         http.MethodGet,
			wantStatusCode: http.StatusOK,
		},
		{
			name:   "Post request - unsupported media type",
			method: http.MethodPost,
			header: http.Header{
				"Content-Type": []string{"test"},
			},
			wantStatusCode: http.StatusUnsupportedMediaType,
		},
		{
			name:   "Post request - supported media type",
			method: http.MethodPut,
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:   "Put request - supported media type",
			method: http.MethodPut,
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:   "Patch request - supported media type",
			method: http.MethodPatch,
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := mux.NewRouter()
			path := "/test"

			GetTestHandler().AddRoute(r, path, test.method)
			r.Use(Body)

			req := httptest.NewRequest(test.method, path, bytes.NewBuffer([]byte("")))

			req.Header = test.header

			r.ServeHTTP(w, req)

			if test.wantStatusCode != w.Code {
				t.Errorf("test %s failed %d != %d", test.name, test.wantStatusCode, w.Code)
			}
		})

	}
}
