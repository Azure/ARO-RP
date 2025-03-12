package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
)

// MockClientAuthorizer mocks ClientAuthorizer for testing.
type MockClientAuthorizer struct {
	isAuthorized bool
}

func (m *MockClientAuthorizer) IsAuthorized(_ *tls.ConnectionState) bool {
	return m.isAuthorized
}

func (m *MockClientAuthorizer) IsReady() bool {
	return true
}

// MockMISEAdapter mocks MISEAdapter for testing.
type MockMISEAdapter struct {
	isAuthorized bool
	err          error
}

func (m *MockMISEAdapter) IsAuthorized(_ context.Context, _ *http.Request) (bool, error) {
	return m.isAuthorized, m.err
}

func (m *MockMISEAdapter) IsReady() bool {
	return true
}

func TestAuthenticate(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name         string
		apiVersion   string
		urlPath      string
		isAdminAuth  bool
		isArmAuth    bool
		isMiseAuth   bool
		enforceMISE  bool
		enableMISE   bool
		expectStatus int
	}{
		{
			name:         "Admin authentication success",
			apiVersion:   admin.APIVersion,
			isAdminAuth:  true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "Admin authentication failure",
			apiVersion:   admin.APIVersion,
			isAdminAuth:  false,
			expectStatus: http.StatusForbidden,
		},
		{
			name:         "ARM authentication success via MISE",
			enableMISE:   true,
			isMiseAuth:   true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "ARM authentication failure via MISE, fallback to TLS success",
			enableMISE:   true,
			isMiseAuth:   false,
			isArmAuth:    true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "ARM authentication failure via MISE, TLS also fails",
			enableMISE:   true,
			isMiseAuth:   false,
			isArmAuth:    false,
			expectStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminAuth := &MockClientAuthorizer{isAuthorized: tt.isAdminAuth}
			armAuth := &MockClientAuthorizer{isAuthorized: tt.isArmAuth}
			miseAuth := &MockMISEAdapter{isAuthorized: tt.isMiseAuth, err: nil}

			middleware := AuthMiddleware{
				Log:         logger,
				EnableMISE:  tt.enableMISE,
				EnforceMISE: tt.enforceMISE,
				AdminAuth:   adminAuth,
				ArmAuth:     armAuth,
				MiseAuth:    miseAuth,
			}

			handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			r := httptest.NewRequest(http.MethodGet, "http://localhost"+tt.urlPath, nil)
			r.URL.RawQuery = api.APIVersionKey + "=" + tt.apiVersion
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			if w.Code != tt.expectStatus {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.expectStatus, w.Code)
			}
		})
	}
}
