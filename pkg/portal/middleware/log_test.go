package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestLog(t *testing.T) {
	h, log := testlog.New()
	ah, auditLog := testlog.NewAudit()
	otelAudit := testlog.NewOtelAuditClient()

	controller := gomock.NewController(t)
	defer controller.Finish()
	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
	_env.EXPECT().Hostname().AnyTimes().Return("testhost")
	_env.EXPECT().Location().AnyTimes().Return("eastus")

	ctx := context.WithValue(context.Background(), ContextKeyUsername, "username")
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://localhost/", strings.NewReader("body"))
	if err != nil {
		t.Fatal(err)
	}
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("User-Agent", "user-agent")

	w := httptest.NewRecorder()

	// chain a custom handler with the Log middleware to mutate the request
	Log(_env, auditLog, log, otelAudit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL = nil // mutate the request

		_ = w.(http.Hijacker) // must implement http.Hijacker

		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, r.Body)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Error(w.Code)
	}

	expected := []map[string]types.GomegaMatcher{
		{
			"msg":                 gomega.Equal("read request"),
			"level":               gomega.Equal(logrus.InfoLevel),
			"request_method":      gomega.Equal("POST"),
			"request_path":        gomega.Equal("/"),
			"request_proto":       gomega.Equal("HTTP/1.1"),
			"request_remote_addr": gomega.Equal("127.0.0.1:1234"),
			"request_user_agent":  gomega.Equal("user-agent"),
			"username":            gomega.Equal("username"),
		},
		{
			"msg":                  gomega.Equal("sent response"),
			"level":                gomega.Equal(logrus.InfoLevel),
			"body_read_bytes":      gomega.Equal(4),
			"body_written_bytes":   gomega.Equal(4),
			"response_status_code": gomega.Equal(http.StatusOK),
			"request_method":       gomega.Equal("POST"),
			"request_path":         gomega.Equal("/"),
			"request_proto":        gomega.Equal("HTTP/1.1"),
			"request_remote_addr":  gomega.Equal("127.0.0.1:1234"),
			"request_user_agent":   gomega.Equal("user-agent"),
			"username":             gomega.Equal("username"),
		},
	}

	err = testlog.AssertLoggingOutput(h, expected)
	if err != nil {
		t.Error(err)
	}

	// only one audit log is associated with the one POST request
	expectedAudit := []*audit.Payload{
		{
			EnvVer:               audit.IFXAuditVersion,
			EnvName:              audit.IFXAuditName,
			EnvFlags:             257,
			EnvAppID:             audit.SourceAdminPortal,
			EnvCloudName:         _env.Environment().Name,
			EnvCloudRole:         audit.CloudRoleRP,
			EnvCloudRoleInstance: _env.Hostname(),
			EnvCloudEnvironment:  _env.Environment().Name,
			EnvCloudLocation:     _env.Location(),
			EnvCloudVer:          audit.IFXAuditCloudVer,
			CallerIdentities: []audit.CallerIdentity{
				{
					CallerIdentityType:  audit.CallerIdentityTypeUsername,
					CallerIdentityValue: "username",
					CallerIPAddress:     "127.0.0.1:1234",
				},
			},
			Category:      "ResourceManagement",
			OperationName: "POST /",
			Result: audit.Result{
				ResultType:        "Success",
				ResultDescription: "Status code: 200",
			},
			TargetResources: []audit.TargetResource{
				{
					TargetResourceType: "",
					TargetResourceName: "/",
				},
			},
		},
	}
	testlog.AssertAuditPayloads(t, ah, expectedAudit)

	for _, e := range h.Entries {
		fmt.Println(e)
	}
}

func TestAuditTargetResourceData(t *testing.T) {
	var (
		testSubscription  = "test-sub"
		testResourceGroup = "test-rg"
		testResource      = "test-resource"
	)

	var testCases = []struct {
		url          string
		expectedType string
		expectedName string
	}{
		{
			url:          "/",
			expectedName: "/",
		},
		{
			url:          "/index.html",
			expectedName: "/index.html",
		},
		{
			url:          "/lib/bootstrap-4.5.2.min.css",
			expectedName: "/lib/bootstrap-4.5.2.min.css",
		},
		{
			url:          "/api/clusters",
			expectedName: "/api/clusters",
		},
		{
			url:          "/api/logout",
			expectedName: "/api/logout",
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/ssh/new", testSubscription, testResourceGroup, testResource),
			expectedName: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/ssh/new", testSubscription, testResourceGroup, testResource),
			expectedType: "ssh",
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/kubeconfig/new", testSubscription, testResourceGroup, testResource),
			expectedName: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/kubeconfig/new", testSubscription, testResourceGroup, testResource),
			expectedType: "kubeconfig",
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/kubeconfig/proxy", testSubscription, testResourceGroup, testResource),
			expectedName: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/kubeconfig/proxy", testSubscription, testResourceGroup, testResource),
			expectedType: "kubeconfig",
		},
		{
			url:          fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/prometheus", testSubscription, testResourceGroup, testResource),
			expectedName: fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s/prometheus", testSubscription, testResourceGroup, testResource),
			expectedType: "prometheus",
		},
	}

	for _, tc := range testCases {
		parsedURL, err := url.Parse(tc.url)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		r := &http.Request{URL: parsedURL}
		if actual := auditTargetResourceType(r); tc.expectedType != actual {
			t.Errorf("%s: expected: %s, actual: %s", tc.url, tc.expectedType, actual)
		}

		if actual := r.URL.Path; tc.expectedName != actual {
			t.Errorf("%s: expected: %s, actual: %s", tc.url, tc.expectedName, actual)
		}
	}
}
