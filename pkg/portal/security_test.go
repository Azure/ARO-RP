package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golangci/golangci-lint/pkg/sliceutil"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/listener"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	"github.com/Azure/ARO-RP/test/util/testpoller"
)

var (
	nonElevatedGroupIDs = []string{"00000000-1111-1111-1111-000000000000"}
	elevatedGroupIDs    = []string{"00000000-0000-0000-0000-000000000000"}
)

func TestSecurity(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	_, portalAccessLog := testlog.New()
	_, portalLog := testlog.New()
	auditHook, portalAuditLog := testlog.NewAudit()

	controller := gomock.NewController(t)
	defer controller.Finish()

	_env := mock_env.NewMockCore(controller)
	_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
	_env.EXPECT().Location().AnyTimes().Return("eastus")
	_env.EXPECT().TenantID().AnyTimes().Return("00000000-0000-0000-0000-000000000001")
	_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
	_env.EXPECT().Hostname().AnyTimes().Return("testhost")

	l := listener.NewListener()
	defer l.Close()

	sshl := listener.NewListener()
	defer sshl.Close()

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	sshkey, _, err := utiltls.GenerateKeyAndCertificate("ssh", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()
	dbPortal, _ := testdatabase.NewFakePortal()

	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	c := &http.Client{
		Transport: &http.Transport{
			DialContext: l.DialContext,
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	p := NewPortal(_env, portalAuditLog, portalLog, portalAccessLog, l, sshl, nil, "", serverkey, servercerts, "", nil, nil, make([]byte, 32), sshkey, nil, elevatedGroupIDs, dbOpenShiftClusters, dbPortal, nil, &noop.Noop{})
	go func() {
		err := p.Run(ctx)
		if err != nil {
			log.Error(err)
		}
	}()

	for _, tt := range []struct {
		name                          string
		request                       func() (*http.Request, error)
		checkResponse                 func(*testing.T, bool, bool, *http.Response)
		unauthenticatedWantStatusCode int
		authenticatedWantStatusCode   int
		wantAuditOperation            string
		wantAuditTargetResources      []audit.TargetResource
	}{
		{
			name: "/",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/", nil)
			},
			unauthenticatedWantStatusCode: 307,
			authenticatedWantStatusCode:   200,
			wantAuditOperation:            "GET /",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "",
					TargetResourceName: "/",
				},
			},
		},
		{
			name: "/main.js",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/main.js", nil)
			},
			wantAuditOperation: "GET /main.js",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "",
					TargetResourceName: "/main.js",
				},
			},
		},
		{
			name: "/api/clusters",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/api/clusters", nil)
			},
			wantAuditOperation: "GET /api/clusters",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "",
					TargetResourceName: "/api/clusters",
				},
			},
		},
		{
			name: "/api/logout",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodPost, "https://server/api/logout", nil)
			},
			unauthenticatedWantStatusCode: http.StatusSeeOther,
			authenticatedWantStatusCode:   http.StatusSeeOther,
			wantAuditOperation:            "POST /api/logout",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "",
					TargetResourceName: "/api/logout",
				},
			},
		},
		{
			name: "/callback",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/callback", nil)
			},
			unauthenticatedWantStatusCode: http.StatusTemporaryRedirect,
			authenticatedWantStatusCode:   http.StatusTemporaryRedirect,
			wantAuditOperation:            "GET /callback",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "",
					TargetResourceName: "/callback",
				},
			},
		},
		{
			name: "/healthz/ready",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/healthz/ready", nil)
			},
			unauthenticatedWantStatusCode: http.StatusOK,
			wantAuditOperation:            "GET /healthz/ready",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "",
					TargetResourceName: "/healthz/ready",
				},
			},
		},
		{
			name: "/kubeconfig/new",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodPost, "https://server/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/resourceName/kubeconfig/new", nil)
			},
			wantAuditOperation: "POST /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/kubeconfig/new",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "kubeconfig",
					TargetResourceName: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/kubeconfig/new",
				},
			},
		},
		{
			name: "/prometheus",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodPost, "https://server/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/resourceName/prometheus", nil)
			},
			authenticatedWantStatusCode: http.StatusTemporaryRedirect,
			wantAuditOperation:          "POST /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/prometheus",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "prometheus",
					TargetResourceName: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/prometheus",
				},
			},
		},
		{
			name: "/ssh/new",
			request: func() (*http.Request, error) {
				req, err := http.NewRequest(http.MethodPost, "https://server/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/resourceName/ssh/new", strings.NewReader("{}"))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Content-Type", "application/json")

				return req, nil
			},
			checkResponse: func(t *testing.T, authenticated, elevated bool, resp *http.Response) {
				if authenticated && !elevated {
					var e struct {
						Error string
					}
					err := json.NewDecoder(resp.Body).Decode(&e)
					if err != nil {
						t.Fatal(err)
					}
					if e.Error != "Elevated access is required." {
						t.Error(e.Error)
					}
				}
			},
			wantAuditOperation: "POST /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/ssh/new",
			wantAuditTargetResources: []audit.TargetResource{
				{
					TargetResourceType: "ssh",
					TargetResourceName: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroupname/providers/microsoft.redhatopenshift/openshiftclusters/resourcename/ssh/new",
				},
			},
		},
		{
			name: "/doesnotexist",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/doesnotexist", nil)
			},
			unauthenticatedWantStatusCode: http.StatusNotFound,
			authenticatedWantStatusCode:   http.StatusNotFound,
		},
	} {
		for _, tt2 := range []struct {
			name           string
			authenticated  bool
			elevated       bool
			wantStatusCode int
		}{
			{
				name:           "unauthenticated",
				wantStatusCode: tt.unauthenticatedWantStatusCode,
			},
			{
				name:           "authenticated",
				authenticated:  true,
				wantStatusCode: tt.authenticatedWantStatusCode,
			},
			{
				name:           "elevated",
				authenticated:  true,
				elevated:       true,
				wantStatusCode: tt.authenticatedWantStatusCode,
			},
		} {
			t.Run(tt2.name+tt.name, func(t *testing.T) {
				defer auditHook.Reset()

				req, err := tt.request()
				if err != nil {
					t.Fatal(err)
				}

				err = addCSRF(req)
				if err != nil {
					t.Fatal(err)
				}

				if tt2.authenticated {
					var groups []string
					if tt2.elevated {
						groups = elevatedGroupIDs
					}
					err = addAuth(req, groups)
					if err != nil {
						t.Fatal(err)
					}
				}

				resp, err := c.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()

				if tt2.wantStatusCode == 0 {
					if tt2.authenticated {
						tt2.wantStatusCode = http.StatusOK
					} else {
						tt2.wantStatusCode = http.StatusTemporaryRedirect
					}
				}

				if resp.StatusCode != tt2.wantStatusCode {
					t.Error(resp.StatusCode, tt2.wantStatusCode)
					body := make([]byte, 0)
					_, err := resp.Body.Read(body)
					if err != nil {
						t.Fatal(err)
					}
					t.Error(body)
				}

				if tt.checkResponse != nil {
					tt.checkResponse(t, tt2.authenticated, tt2.elevated, resp)
				}

				// no audit logs for https://server/doesnotexist
				if tt.authenticatedWantStatusCode == http.StatusNotFound {
					return
				}

				// perform some polling on static files because the http.ServeContent() calls in the
				// portal's serve() and index() handlers[1] issued a call to io.Copy()[2]
				// causes a race condition with the audit hook. The response was returned
				// to the client and the testlog.AssertAuditPayloads() was called immediately,
				// while the audit hook was still in-flight.
				//
				// note that the audit logs will still be recorded and emitted by the audit
				// hook, so this is a non-issue in the Geneva environment.
				//
				// [1] https://github.com/Azure/ARO-RP/blob/master/pkg/portal/portal.go#L222-L247
				// [2] https://go.googlesource.com/go/+/go1.16.2/src/net/http/fs.go#337
				//
				// TODO: there is a data race that exists only within this test independent of the polling
				// race mentioned above. AllEntries returns a copy of the current entries within logrus,
				// but the underlying data within the entry is not copied over.  When we attempt to
				// get the entry in the Data map for the MetadataPayload, there is a slight chance that
				// the Payload will change during this access, resulting in the e2e panicking.
				// `go test -race -timeout 30s -run ^TestSecurity$ ./pkg/portal` should show the race and
				// where the concurrent read/write is occurring.
				if tt.name == "/" || tt.name == "/main.js" {
					err = testpoller.Poll(1*time.Second, 5*time.Millisecond, func() (bool, error) {
						if len(auditHook.AllEntries()) == 1 {
							if _, ok := auditHook.AllEntries()[0].Data[audit.MetadataPayload]; ok {
								return true, nil
							}
						}
						return false, nil
					})
					if err != nil {
						t.Error(err)
					}
				}

				if tt.wantAuditOperation != "" {
					payload := auditPayloadFixture()
					payload.OperationName = tt.wantAuditOperation
					payload.TargetResources = tt.wantAuditTargetResources
					payload.Result.ResultDescription = fmt.Sprintf("Status code: %d", tt2.wantStatusCode)

					if tt2.wantStatusCode == http.StatusForbidden {
						payload.Result.ResultType = audit.ResultTypeFail
					}

					if tt2.authenticated && !sliceutil.Contains([]string{
						"/callback", "/healthz/ready", "/api/login", "/api/logout"}, tt.name) {
						payload.CallerIdentities[0].CallerIdentityValue = "username"
					}
					testlog.AssertAuditPayloads(t, auditHook, []*audit.Payload{payload})
				} else {
					testlog.AssertAuditPayloads(t, auditHook, []*audit.Payload{})
				}
			})
		}
	}
}

func addCSRF(req *http.Request) error {
	if req.Method != http.MethodPost {
		return nil
	}

	req.Header.Set("X-CSRF-Token", base64.StdEncoding.EncodeToString(make([]byte, 64)))

	sc := securecookie.New(make([]byte, 32), nil)
	sc.SetSerializer(securecookie.JSONEncoder{})

	cookie, err := sc.Encode("_gorilla_csrf", make([]byte, 32))
	if err != nil {
		return err
	}
	req.Header.Add("Cookie", "_gorilla_csrf="+cookie)

	return nil
}

func addAuth(req *http.Request, groups []string) error {
	store := sessions.NewCookieStore(make([]byte, 32))

	cookie, err := securecookie.EncodeMulti(middleware.SessionName, map[interface{}]interface{}{
		middleware.SessionKeyUsername: "username",
		middleware.SessionKeyGroups:   groups,
		middleware.SessionKeyExpires:  time.Now().Add(time.Hour).Unix(),
	}, store.Codecs...)
	if err != nil {
		return err
	}
	req.Header.Add("Cookie", middleware.SessionName+"="+cookie)

	return nil
}

func auditPayloadFixture() *audit.Payload {
	return &audit.Payload{
		EnvVer:               audit.IFXAuditVersion,
		EnvName:              audit.IFXAuditName,
		EnvFlags:             257,
		EnvAppID:             audit.SourceAdminPortal,
		EnvCloudName:         azureclient.PublicCloud.Name,
		EnvCloudRole:         audit.CloudRoleRP,
		EnvCloudRoleInstance: "testhost",
		EnvCloudEnvironment:  azureclient.PublicCloud.Name,
		EnvCloudLocation:     "eastus",
		EnvCloudVer:          1,
		CallerIdentities: []audit.CallerIdentity{
			{
				CallerDisplayName:  "",
				CallerIdentityType: audit.CallerIdentityTypeUsername,
				CallerIPAddress:    "bufferedpipe",
			},
		},
		Category: audit.CategoryResourceManagement,
		Result: audit.Result{
			ResultType: audit.ResultTypeSuccess,
		},
	}
}
