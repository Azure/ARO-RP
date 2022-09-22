package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest"
)

type fakeClient struct {
	t          *testing.T
	wantMethod string
	wantURL    string
	resp       *http.Response
	err        error
}

func (fc *fakeClient) Do(req *http.Request) (*http.Response, error) {
	if req.Method != fc.wantMethod {
		fc.t.Fatal(req.Method)
	}

	if req.URL.String() != fc.wantURL {
		fc.t.Fatal(req.URL.String())
	}

	return fc.resp, fc.err
}

func TestClient(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name       string
		fakeClient *fakeClient
		wantToken  string
		wantErr    string
	}{
		{
			name: "works",
			fakeClient: &fakeClient{
				wantMethod: http.MethodPost,
				wantURL:    "https://localhost/token?permission=permission",
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: io.NopCloser(strings.NewReader(`{"token":"token"}`)),
				},
			},
			wantToken: "token",
		},
		{
			name: "404",
			fakeClient: &fakeClient{
				wantMethod: http.MethodPost,
				wantURL:    "https://localhost/token?permission=permission",
				resp: &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
			wantErr: "unexpected status code 404",
		},
		{
			name: "no content-type",
			fakeClient: &fakeClient{
				wantMethod: http.MethodPost,
				wantURL:    "https://localhost/token?permission=permission",
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
			wantErr: `unexpected content type ""`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.fakeClient.t = t

			c := &client{
				c:          tt.fakeClient,
				authorizer: &autorest.NullAuthorizer{},
				url:        "https://localhost",
			}

			token, err := c.Token(ctx, "permission")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Fatal(err)
			}

			if token != tt.wantToken {
				t.Error(token)
			}
		})
	}
}
