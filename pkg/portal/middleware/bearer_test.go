package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestBearer(t *testing.T) {
	for _, tt := range []struct {
		name              string
		fixture           func(*testdatabase.Fixture)
		request           func() (*http.Request, error)
		wantAuthenticated bool
		wantUsername      string
	}{
		{
			name: "authenticated",
			fixture: func(fixture *testdatabase.Fixture) {
				fixture.AddPortalDocuments(&api.PortalDocument{
					ID: "00000000-0000-0000-0000-000000000000",
					Portal: &api.Portal{
						Username: "username",
					},
				})
			},
			request: func() (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{"Bearer 00000000-0000-0000-0000-000000000000"},
					},
				}, nil
			},
			wantAuthenticated: true,
			wantUsername:      "username",
		},
		{
			name: "not authenticated - no header",
			request: func() (*http.Request, error) {
				return &http.Request{}, nil
			},
		},
		{
			name: "not authenticated - bad header",
			request: func() (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{"Bearer bad"},
					},
				}, nil
			},
		},
		{
			name: "not authenticated - berarer not found",
			fixture: func(fixture *testdatabase.Fixture) {
				fixture.AddPortalDocuments(&api.PortalDocument{
					ID: "00000000-0000-0000-0000-000000000000",
					Portal: &api.Portal{
						Username: "username",
					},
				})
			},
			request: func() (*http.Request, error) {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{"Bearer 10000000-0000-0000-0000-000000000000"},
					},
				}, nil
			},
		},
	} {
		dbPortal, _ := testdatabase.NewFakePortal()

		fixture := testdatabase.NewFixture().
			WithPortal(dbPortal)

		if tt.fixture != nil {
			tt.fixture(fixture)
		}

		err := fixture.Create()
		if err != nil {
			t.Fatal(err)
		}

		var username string
		var usernameok bool
		var portaldoc *api.PortalDocument
		var portaldocok bool
		h := Bearer(dbPortal)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, usernameok = r.Context().Value(ContextKeyUsername).(string)
			portaldoc, portaldocok = r.Context().Value(ContextKeyPortalDoc).(*api.PortalDocument)
		}))

		r, err := tt.request()
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()

		h.ServeHTTP(w, r)

		if username != tt.wantUsername {
			t.Error(username)
		}
		if usernameok != tt.wantAuthenticated {
			t.Error(usernameok)
		}
		if portaldocok != tt.wantAuthenticated {
			t.Error(portaldocok)
		}
		if tt.wantAuthenticated && portaldoc.Portal.Username != username {
			t.Error(portaldoc.Portal.Username)
		}
	}
}
