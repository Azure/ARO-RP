package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestPortalInfo(t *testing.T) {
	var tests = []struct {
		name               string
		expected           PortalInfo
		authenticated      bool
		elevated           bool
		expectedStatusCode int
	}{
		{
			name:               "basic",
			authenticated:      true,
			elevated:           false,
			expectedStatusCode: 200,
			expected: PortalInfo{
				Location:  "eastus",
				Username:  "username",
				Elevated:  false,
				RPVersion: version.GitCommit,
			},
		},
		{
			name:               "elevated",
			authenticated:      true,
			elevated:           true,
			expectedStatusCode: 200,
			expected: PortalInfo{
				Location:  "eastus",
				Username:  "username",
				Elevated:  true,
				RPVersion: version.GitCommit,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockCore(controller)
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
			_env.EXPECT().Location().AnyTimes().Return("eastus")
			_env.EXPECT().TenantID().AnyTimes().Return("00000000-0000-0000-0000-000000000001")
			_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			_env.EXPECT().Hostname().AnyTimes().Return("testhost")

			p := NewPortal(_env, nil, nil, nil, nil, nil, nil, "", nil, nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

			request, err := http.NewRequest(http.MethodGet, "/api/info", nil)
			if err != nil {
				t.Fatal("could not create the request ", err)
			}
			request = request.WithContext(context.WithValue(ctx, middleware.ContextKeyUsername, "username"))
			request = request.WithContext(context.WithValue(request.Context(), middleware.ContextKeyGroups, []string{}))
			if tt.elevated {
				request = request.WithContext(context.WithValue(request.Context(), middleware.ContextKeyGroups, []string{"elevated"}))
			}

			writer := httptest.NewRecorder()
			portal := p.(*portal)
			portal.elevatedGroupIDs = []string{"elevated"}
			portal.info(writer, request)
			if writer.Result().StatusCode != tt.expectedStatusCode {
				t.Errorf("%d != %d", writer.Result().StatusCode, tt.expectedStatusCode)
			}

			if writer.Result().Header.Get("Content-Type") != "application/json" {
				t.Error(writer.Result().Header.Get("Content-Type"))
			}

			var readResp PortalInfo
			err = json.NewDecoder(writer.Result().Body).Decode(&readResp)
			if err != nil {
				t.Fatal(err)
			}

			// copy through the CSRF token if it's non-blank, since we can't make it
			// a known value
			if readResp.CSRFToken != "" {
				tt.expected.CSRFToken = readResp.CSRFToken
			}

			for _, l := range deep.Equal(readResp, tt.expected) {
				t.Error(l)
			}
		})
	}
}
