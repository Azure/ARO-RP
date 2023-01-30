package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	mock_cosmosdb "github.com/Azure/ARO-RP/pkg/util/mocks/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/oidc"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name                    string
		permissionClientFactory func(controller *gomock.Controller) func(userid string) cosmosdb.PermissionClient
		req                     *http.Request
		wantStatusCode          int
		wantToken               string
	}{
		{
			name: "GET /random returns 404",
			req: &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Scheme: "http",
					Host:   "localhost",
					Path:   "/random",
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "GET /healthz/ready returns 200",
			req: &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Scheme: "http",
					Host:   "localhost",
					Path:   "/healthz/ready",
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "GET unauthorized /token returns 403",
			req: &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Scheme: "http",
					Host:   "localhost",
					Path:   "/token",
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "POST /token?permission=good returns 403 (no auth)",
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=good",
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "POST /token?permission=good returns 403 (empty subject)",
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=good",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": ""}`},
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "POST /token?permission=good returns 403 (subject not UUID)",
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=good",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": "xyz"}`},
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "POST /token returns 400",
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme: "http",
					Host:   "localhost",
					Path:   "/token",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": "00000000-0000-0000-0000-000000000000"}`},
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "POST /token?permission=bad! returns 400",
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=bad!",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": "00000000-0000-0000-0000-000000000000"}`},
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "POST /token?permission=notexist returns 400",
			permissionClientFactory: func(controller *gomock.Controller) func(userid string) cosmosdb.PermissionClient {
				return func(userid string) cosmosdb.PermissionClient {
					permc := mock_cosmosdb.NewMockPermissionClient(controller)
					permc.EXPECT().Get(gomock.Any(), "notexist").Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
					return permc
				}
			},
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=notexist",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": "00000000-0000-0000-0000-000000000000"}`},
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "POST /token?permission=perm and database error returns 500",
			permissionClientFactory: func(controller *gomock.Controller) func(userid string) cosmosdb.PermissionClient {
				return func(userid string) cosmosdb.PermissionClient {
					permc := mock_cosmosdb.NewMockPermissionClient(controller)
					permc.EXPECT().Get(gomock.Any(), "perm").Return(nil, errors.New("sad database"))
					return permc
				}
			},
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=perm",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": "00000000-0000-0000-0000-000000000000"}`},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "get /token?permission=perm returns 405",
			permissionClientFactory: func(controller *gomock.Controller) func(userid string) cosmosdb.PermissionClient {
				return func(userid string) cosmosdb.PermissionClient {
					permc := mock_cosmosdb.NewMockPermissionClient(controller)
					permc.EXPECT().Get(gomock.Any(), "perm").Return(&cosmosdb.Permission{
						Token: "token",
					}, nil)
					return permc
				}
			},
			req: &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=perm",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": "00000000-0000-0000-0000-000000000000"}`},
				},
			},
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name: "POST /token?permission=perm returns 200",
			permissionClientFactory: func(controller *gomock.Controller) func(userid string) cosmosdb.PermissionClient {
				return func(userid string) cosmosdb.PermissionClient {
					permc := mock_cosmosdb.NewMockPermissionClient(controller)
					permc.EXPECT().Get(gomock.Any(), "perm").Return(&cosmosdb.Permission{
						Token: "token",
					}, nil)
					return permc
				}
			},
			req: &http.Request{
				Method: http.MethodPost,
				URL: &url.URL{
					Scheme:   "http",
					Host:     "localhost",
					Path:     "/token",
					RawQuery: "permission=perm",
				},
				Header: http.Header{
					"Authorization": []string{`Bearer {"sub": "00000000-0000-0000-0000-000000000000"}`},
				},
			},
			wantStatusCode: http.StatusOK,
			wantToken:      "token",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			l := listener.NewListener()
			defer l.Close()

			s := &server{
				log:       logrus.NewEntry(logrus.StandardLogger()),
				accessLog: logrus.NewEntry(logrus.StandardLogger()),
				l:         l,
				verifier:  &oidc.NoopVerifier{},
			}

			if tt.permissionClientFactory != nil {
				s.permissionClientFactory = tt.permissionClientFactory(controller)
			}

			go func() {
				_ = s.Run(ctx)
			}()

			c := &http.Client{
				Transport: &http.Transport{
					DialContext: l.DialContext,
				},
			}

			resp, err := c.Do(tt.req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			if tt.wantToken == "" {
				return
			}

			if resp.Header.Get("Content-Type") != "application/json" {
				t.Fatal(resp.Header.Get("Content-Type"))
			}

			var tr *tokenResponse
			err = json.NewDecoder(resp.Body).Decode(&tr)
			if err != nil {
				t.Fatal(err)
			}

			if tr.Token != tt.wantToken {
				t.Error(tr.Token)
			}
		})
	}
}
