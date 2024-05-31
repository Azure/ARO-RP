package api_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
	"net/http"
	"testing"
)

func TestRequestBuilder_Build(t *testing.T) {
	testCases := []struct {
		name     string
		method   string
		baseURL  string
		endpoint string
		headers  map[string]string
		params   map[string]string
		body     []byte
		expURL   string
		expErr   bool
	}{
		{
			name:     "Test Request Builder",
			method:   http.MethodGet,
			baseURL:  "http://example.com",
			endpoint: "/api/v1/test",
			headers:  map[string]string{"Content-Type": "application/json"},
			params:   map[string]string{"key": "value"},
			body:     []byte(`{"key":"value"}`),
			expURL:   "http://example.com/api/v1/test?key=value",
			expErr:   false,
		},
		{
			name:     "Test Invalid URL",
			method:   http.MethodGet,
			baseURL:  "://invalid.url",
			endpoint: "/api/v1/test",
			headers:  map[string]string{"Content-Type": "application/json"},
			params:   map[string]string{"key": "value"},
			body:     []byte(`{"key":"value"}`),
			expErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rb := api.NewRequestBuilder(tc.method, tc.baseURL).
				SetEndpoint(tc.endpoint).
				SetBody(tc.body)

			for key, value := range tc.headers {
				rb.AddHeader(key, value)
			}

			for key, value := range tc.params {
				rb.AddParam(key, value)
			}

			req, err := rb.Build()
			if (err != nil) != tc.expErr {
				t.Fatalf("RequestBuilder Build error %v, expected error %v", err, tc.expErr)
				return
			}

			if tc.expErr {
				return
			}

			gotURL := req.URL.String()
			if gotURL != tc.expURL {
				t.Errorf("Got URL %v, expect %v", gotURL, tc.expURL)
			}

			for key, value := range tc.headers {
				gotValue := req.Header.Get(key)
				if gotValue != value {
					t.Errorf("Got Header %v = %v, expect %v", key, gotValue, value)
				}
			}
		})
	}
}
