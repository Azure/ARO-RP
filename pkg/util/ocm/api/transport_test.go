package api_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/ocm/api"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessTokenTransport_RoundTrip(t *testing.T) {
	testCases := []struct {
		name          string
		authToken     *api.AccessToken
		expAuthHeader string
	}{
		{
			name:          "Test Authorization header",
			authToken:     api.NewAccessToken("test", "123"),
			expAuthHeader: "AccessToken test:123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Test the Auth header
				authHeader := r.Header.Get("Authorization")
				if authHeader != tc.expAuthHeader {
					t.Errorf("Authorization header = %v, expect %v", authHeader, tc.expAuthHeader)
				}
			}))
			defer server.Close()

			accessTokenTransport := api.NewAccessTokenTransport(tc.authToken)

			req := httptest.NewRequest(http.MethodGet, server.URL, nil)
			_, err := accessTokenTransport.RoundTrip(req)
			if err != nil {
				t.Fatalf("AccessTokenTransport RoundTrip error = %v", err)
			}
		})
	}
}
