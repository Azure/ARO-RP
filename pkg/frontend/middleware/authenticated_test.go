package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/util/miseadapter"
	testlog "github.com/Azure/ARO-RP/test/util/log"
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

func TestAuthenticate(t *testing.T) {
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
		expectLogs   []testlog.ExpectedLogEntry
	}{
		{
			name:         "Admin authentication success",
			apiVersion:   admin.APIVersion,
			isAdminAuth:  true,
			expectStatus: http.StatusOK,
			expectLogs:   []testlog.ExpectedLogEntry{},
		},
		{
			name:         "Admin authentication failure",
			apiVersion:   admin.APIVersion,
			isAdminAuth:  false,
			expectStatus: http.StatusForbidden,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("Authentication Failed"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication success via MISE",
			enableMISE:   true,
			isMiseAuth:   true,
			expectStatus: http.StatusOK,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.InfoLevel),
					"msg":       gomega.Equal("MISE authorization successful"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication failure via MISE, fallback to TLS success",
			enableMISE:   true,
			isMiseAuth:   false,
			isArmAuth:    true,
			expectStatus: http.StatusOK,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful, enforcing: false, error: %!s(<nil>)"),
					"requestID": gomega.Equal("1234"),
				},
				{
					"level":     gomega.Equal(logrus.WarnLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful/disabled, fallback to TLS certificate authentication"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication failure via MISE, TLS also fails",
			enableMISE:   true,
			isMiseAuth:   false,
			isArmAuth:    false,
			expectStatus: http.StatusForbidden,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful, enforcing: false, error: %!s(<nil>)"),
					"requestID": gomega.Equal("1234"),
				},
				{
					"level":     gomega.Equal(logrus.WarnLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful/disabled, fallback to TLS certificate authentication"),
					"requestID": gomega.Equal("1234"),
				},
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("Authentication Failed"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication MISE enforced - success",
			enableMISE:   true,
			enforceMISE:  true,
			isMiseAuth:   true,
			isArmAuth:    false,
			expectStatus: http.StatusOK,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.InfoLevel),
					"msg":       gomega.Equal("MISE authorization successful"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication MISE enforced - failure, no TLS fallback",
			enableMISE:   true,
			enforceMISE:  true,
			isMiseAuth:   false,
			isArmAuth:    true, // TLS would succeed but should be ignored
			expectStatus: http.StatusForbidden,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful, enforcing: true, error: %!s(<nil>)"),
					"requestID": gomega.Equal("1234"),
				},
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("Authentication Failed"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication MISE enforced - both would fail",
			enableMISE:   true,
			enforceMISE:  true,
			isMiseAuth:   false,
			isArmAuth:    false,
			expectStatus: http.StatusForbidden,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful, enforcing: true, error: %!s(<nil>)"),
					"requestID": gomega.Equal("1234"),
				},
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("Authentication Failed"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication MISE disabled, TLS success",
			enableMISE:   false,
			enforceMISE:  false,
			isMiseAuth:   false,
			isArmAuth:    true,
			expectStatus: http.StatusOK,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.WarnLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful/disabled, fallback to TLS certificate authentication"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
		{
			name:         "ARM authentication MISE disabled, TLS failure",
			enableMISE:   false,
			enforceMISE:  false,
			isMiseAuth:   false,
			isArmAuth:    false,
			expectStatus: http.StatusForbidden,
			expectLogs: []testlog.ExpectedLogEntry{
				{
					"level":     gomega.Equal(logrus.WarnLevel),
					"msg":       gomega.Equal("MISE authorization unsuccessful/disabled, fallback to TLS certificate authentication"),
					"requestID": gomega.Equal("1234"),
				},
				{
					"level":     gomega.Equal(logrus.ErrorLevel),
					"msg":       gomega.Equal("Authentication Failed"),
					"requestID": gomega.Equal("1234"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminAuth := &MockClientAuthorizer{isAuthorized: tt.isAdminAuth}
			armAuth := &MockClientAuthorizer{isAuthorized: tt.isArmAuth}
			miseAuth := miseadapter.NewFakeAuthorizer(true, tt.isMiseAuth)

			hook, log := testlog.LogForTesting(t)

			middleware := AuthMiddleware{
				Log:         log,
				EnableMISE:  tt.enableMISE,
				EnforceMISE: tt.enforceMISE,
				AdminAuth:   adminAuth,
				ArmAuth:     armAuth,
				MiseAuth:    miseAuth,
			}

			handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create a logger which we attach onto the request context similar
			// to how the log middleware would, and then we can check for the
			// log key set
			requestLog := log.WithField("requestID", "1234")
			reqctx := context.WithValue(context.Background(), ContextKeyLog, requestLog)

			r := httptest.NewRequestWithContext(reqctx, http.MethodGet, "http://localhost"+tt.urlPath, nil)
			r.URL.RawQuery = api.APIVersionKey + "=" + tt.apiVersion
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			if w.Code != tt.expectStatus {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.expectStatus, w.Code)
			}

			err := testlog.AssertLoggingOutput(hook, tt.expectLogs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
