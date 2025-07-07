package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/test/validate"
)

func TestBody(t *testing.T) {
	tests := []struct {
		name    string
		isGet   bool
		header  http.Header
		body    []byte
		wantErr string
	}{
		{
			name:  "GET request - valid",
			isGet: true,
		},
		{
			name:    "non-GET request - large body",
			body:    bytes.Repeat([]byte{0}, 1048577),
			wantErr: "400: InvalidResource: : The resource definition is invalid.",
		},
		{
			name: "non-GET request - invalid media type",
			header: http.Header{
				"Content-Type": []string{"invalid"},
			},
			wantErr: "415: UnsupportedMediaType: : The content media type 'invalid' is not supported. Only 'application/json' is supported.",
		},
		{
			name: "non-GET request - empty media type allowed with empty body",
		},
		{
			name:    "non-GET request - empty media type not allowed with non-empty body",
			body:    []byte("body"),
			wantErr: "415: UnsupportedMediaType: : The content media type '' is not supported. Only 'application/json' is supported.",
		},
		{
			name: "non-GET request - valid media type allowed with empty body",
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "non-GET request - valid media type allowed with non-empty body",
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			body: []byte("body"),
		},
	}

	for _, tt := range tests {
		methods := []string{http.MethodGet}
		if !tt.isGet {
			methods = []string{http.MethodPatch, http.MethodPost, http.MethodPatch}
		}

		for _, method := range methods {
			t.Run(tt.name+"/"+method, func(t *testing.T) {
				r, err := http.NewRequest(method, "", bytes.NewReader(tt.body))
				if err != nil {
					t.Fatal(err)
				}
				r.Header = tt.header

				w := httptest.NewRecorder()

				Body(http.HandlerFunc(func(w http.ResponseWriter, _r *http.Request) {
					r = _r
				})).ServeHTTP(w, r)

				if tt.wantErr == "" {
					if w.Code != http.StatusOK {
						t.Error(w.Code)
					}

					if w.Body.String() != "" {
						t.Error(w.Body.String())
					}

					if !tt.isGet {
						body := r.Context().Value(ContextKeyBody).([]byte)
						if !bytes.Equal(body, tt.body) {
							t.Error(string(body))
						}
					}
				} else {
					var cloudErr *api.CloudError
					err = json.Unmarshal(w.Body.Bytes(), &cloudErr)
					if err != nil {
						t.Fatal(err)
					}
					cloudErr.StatusCode = w.Code

					validate.CloudError(t, cloudErr)

					if tt.wantErr != cloudErr.Error() {
						t.Error(cloudErr)
					}
				}
			})
		}
	}
}
