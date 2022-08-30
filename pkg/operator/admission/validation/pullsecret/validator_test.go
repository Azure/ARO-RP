package pullsecret

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	mock_pullsecretadmission "github.com/Azure/ARO-RP/pkg/util/mocks/pullsecretadmission"
)

func TestExtractFromHeader(t *testing.T) {
	for _, tt := range []struct {
		name            string
		input           string
		wantBearerRealm string
		wantService     string
		wantErr         error
	}{
		{
			name:            "ok",
			input:           `Bearer realm="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",service="docker-registry"`,
			wantBearerRealm: "https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",
			wantService:     "docker-registry",
		},
		{
			name:    "malformed header",
			input:   `="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",service="docker-r egistry"`,
			wantErr: fmt.Errorf("header is missing data"),
		},
		{
			name:    "missing service",
			input:   `Bearer realm="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth"`,
			wantErr: fmt.Errorf("header is missing data"),
		},
		{
			name:    "missing service 2",
			input:   `Bearer realm="https://registry.connect.redhat.com/auth/realms/rhcc/protocol/redhat-docker-v2/auth",`,
			wantErr: fmt.Errorf("header is missing data"),
		},
	} {
		t.Run(tt.name, func(*testing.T) {
			realm, service, err := extractValuesFromAuthHeader(tt.input)
			if realm != tt.wantBearerRealm {
				t.Error(tt.name)
			}
			if service != tt.wantService {
				t.Error(tt.name)
			}
			if err != nil && tt.wantErr == nil {
				t.Error(tt.name)
			} else if err != nil && err.Error() != tt.wantErr.Error() {
				t.Error(tt.name)
			}
		})
	}
}

func TestFetchAuthUrl(t *testing.T) {
	for _, tt := range []struct {
		name        string
		statusCode  int
		returnedErr error
		wantErr     string
	}{
		{
			name:       "wrong status",
			statusCode: 0,
			wantErr: fmt.Sprintf("unexpected status code for https://%s/v2 , got %d but expected %d or %d",
				"", 0, http.StatusOK, http.StatusUnauthorized),
		},
		{
			name:        "request error",
			statusCode:  -1,
			returnedErr: fmt.Errorf("some error"),
			wantErr:     "some error",
		},
		//headers are not set, but tested when tested extractValuesFromAuthHeader
		{
			name:       "200",
			statusCode: 200,
			wantErr:    "header is missing data",
		},
		//headers are not set, but tested when tested extractValuesFromAuthHeader
		{
			name:       "401",
			statusCode: 401,
			wantErr:    "header is missing data",
		},
	} {
		t.Run(tt.name, func(*testing.T) {
			log := logrus.New()
			log.SetOutput(ioutil.Discard)
			entry := logrus.NewEntry(log)
			mockController := gomock.NewController(t)
			defer mockController.Finish()
			requestDoerMock := mock_pullsecretadmission.NewMockRequestDoer(mockController)

			requestDoerMock.
				EXPECT().
				Do(gomock.Any()).
				AnyTimes().
				Return(
					&http.Response{
						StatusCode: tt.statusCode,
					},
					tt.returnedErr)

			client := ociRegClient{httpClient: requestDoerMock, ctx: context.Background()}
			_, _, err := client.fetchAuthURL(entry, "")
			if err.Error() != tt.wantErr {
				t.Errorf("fetchAuthURL failed, wanted err=%s but got err=%s", tt.wantErr, err)
			}
		})
	}
}

func TestGetToken(t *testing.T) {
	for _, tt := range []struct {
		name        string
		statusCode  int
		returnedErr error
		wantErr     string
		body        io.Reader
		wantToken   string
	}{
		{
			name:       "wrong status",
			statusCode: 500,
			wantErr:    fmt.Sprintf("authentication unsucessful, got status code %d but wanted %d", 500, http.StatusOK),
		},
		{
			name:        "response error",
			statusCode:  500,
			returnedErr: fmt.Errorf("some error"),
			wantErr:     "some error",
		},
		{
			name:       "success",
			statusCode: 200,
			body:       strings.NewReader(`{"Token":"supertoken"}`),
			wantToken:  "supertoken",
		},
		{
			name:       "wrong json",
			statusCode: 200,
			body:       strings.NewReader(`{"wrongjson":"notcorrect"}`),
			wantToken:  "",
			wantErr:    "no token in response",
		},
	} {
		t.Run(tt.name, func(*testing.T) {
			log := logrus.New()
			log.SetOutput(ioutil.Discard)
			entry := logrus.NewEntry(log)
			mockController := gomock.NewController(t)
			defer mockController.Finish()
			requestDoerMock := mock_pullsecretadmission.NewMockRequestDoer(mockController)

			requestDoerMock.
				EXPECT().
				Do(gomock.Any()).
				AnyTimes().
				Return(
					&http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(tt.body),
					},
					tt.returnedErr)

			client := ociRegClient{httpClient: requestDoerMock, ctx: context.Background()}
			token, err := client.getToken(entry, "", "", "user", "password")
			if err == nil && tt.wantErr != "" {
				t.Errorf("gettoken failed, wanted err=%s but got err=%s", tt.wantErr, err)
			}
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("gettoken failed, wanted err=%s but got err=%s", tt.wantErr, err)
			}
			if token != tt.wantToken {
				t.Errorf("token was %s but expected %s", token, tt.wantToken)
			}
		})
	}
}
