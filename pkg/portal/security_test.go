package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/listener"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

var nonElevatedGroupIDs = []string{"00000000-1111-1111-1111-000000000000"}

func TestSecurity(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.LogForTesting(t)

	_, portalAccessLog := testlog.New()
	_, portalLog := testlog.New()
	auditHook, portalAuditLog := testlog.NewAudit()
	otelAudit := testlog.NewOtelAuditClient()

	controller := gomock.NewController(t)

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

	dbg := database.NewDBGroup().
		WithOpenShiftClusters(dbOpenShiftClusters).
		WithPortal(dbPortal)

	p := NewPortal(_env, portalAuditLog, portalLog, portalAccessLog, otelAudit, l, sshl, nil, "", serverkey, servercerts, "", nil, nil, make([]byte, 32), sshkey, nil, dbg, nil, &noop.Noop{})
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
			unauthenticatedWantStatusCode: 404,
			authenticatedWantStatusCode:   404,
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
		} {
			t.Run(tt2.name+tt.name, func(t *testing.T) {
				defer auditHook.Reset()

				req, err := tt.request()
				if err != nil {
					t.Fatal(err)
				}

				// Add Referer header for HTTPS otherwise securecookie is unhappy
				req.Header.Add("Referer", "https://server/")

				err = addCSRF(req)
				if err != nil {
					t.Fatal(err)
				}

				if tt2.authenticated {
					err = addAuth(req, []string{})
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

				// Every matched portal response must carry the hardening
				// headers set by the SecurityHeaders middleware.  Unmatched
				// routes (e.g. /) are served by mux's 404 handler,
				// which the middleware chain does not run, so skip those.
				if tt.name != "/" {
					assertSecurityHeaders(t, resp)
				}

				// no audit logs for 404s
				if tt.authenticatedWantStatusCode == http.StatusNotFound {
					return
				}

				if tt.wantAuditOperation != "" {
					payload := auditPayloadFixture()
					payload.OperationName = tt.wantAuditOperation
					payload.TargetResources = tt.wantAuditTargetResources
					payload.Result.ResultDescription = fmt.Sprintf("Status code: %d", tt2.wantStatusCode)

					if tt2.wantStatusCode == http.StatusForbidden {
						payload.Result.ResultType = audit.ResultTypeFail
					}

					if tt2.authenticated && !slices.Contains([]string{
						"/callback", "/healthz/ready", "/api/login", "/api/logout",
					}, tt.name) {
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

func assertSecurityHeaders(t *testing.T, resp *http.Response) {
	t.Helper()

	csp := resp.Header.Get("Content-Security-Policy")
	// script-src must be strict (no 'unsafe-inline') so that cluster-authored
	// inline <script> and event-handler attributes cannot execute on the portal
	// origin.
	if !strings.Contains(csp, "script-src 'self' 'nonce-") {
		t.Errorf("Content-Security-Policy missing strict nonce-based script-src: %q", csp)
	}
	if strings.Contains(csp, "script-src") && strings.Contains(csp, "'unsafe-inline'") &&
		!strings.Contains(csp, "style-src 'self' 'unsafe-inline'") {
		t.Errorf("Content-Security-Policy unexpectedly allows unsafe-inline for script: %q", csp)
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("Content-Security-Policy missing frame-ancestors 'none': %q", csp)
	}

	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
	}

	if got := resp.Header.Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want %q", got, "no-referrer")
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
