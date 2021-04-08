package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/listener"
)

var (
	elevatedGroupIDs = []string{"00000000-0000-0000-0000-000000000000"}
)

func TestSecurity(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	controller := gomock.NewController(t)
	defer controller.Finish()

	_env := mock_env.NewMockCore(controller)
	_env.EXPECT().IsDevelopmentMode().AnyTimes().Return(false)
	_env.EXPECT().Location().AnyTimes().Return("eastus")
	_env.EXPECT().TenantID().AnyTimes().Return("00000000-0000-0000-0000-000000000001")

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

	p := NewPortal(_env, log, log, l, sshl, nil, "", serverkey, servercerts, "", nil, nil, make([]byte, 32), sshkey, nil, elevatedGroupIDs, dbOpenShiftClusters, dbPortal, nil)
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
	}{
		{
			name: "/",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/", nil)
			},
		},
		{
			name: "/index.js",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/index.js", nil)
			},
		},
		{
			name: "/api/clusters",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/api/clusters", nil)
			},
		},
		{
			name: "/api/logout",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodPost, "https://server/api/logout", nil)
			},
			authenticatedWantStatusCode: http.StatusSeeOther,
		},
		{
			name: "/callback",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/callback", nil)
			},
			authenticatedWantStatusCode: http.StatusTemporaryRedirect,
		},
		{
			name: "/healthz/ready",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodGet, "https://server/healthz/ready", nil)
			},
			unauthenticatedWantStatusCode: http.StatusOK,
		},
		{
			name: "/kubeconfig/new",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodPost, "https://server/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/resourceName/kubeconfig/new", nil)
			},
		},
		{
			name: "/prometheus",
			request: func() (*http.Request, error) {
				return http.NewRequest(http.MethodPost, "https://server/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroupName/providers/microsoft.redhatopenshift/openshiftclusters/resourceName/prometheus", nil)
			},
			authenticatedWantStatusCode: http.StatusTemporaryRedirect,
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
					t.Error(resp.StatusCode)
				}

				if tt.checkResponse != nil {
					tt.checkResponse(t, tt2.authenticated, tt2.elevated, resp)
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
		middleware.SessionKeyExpires:  time.Now().Add(time.Hour),
	}, store.Codecs...)
	if err != nil {
		return err
	}
	req.Header.Add("Cookie", middleware.SessionName+"="+cookie)

	return nil
}
