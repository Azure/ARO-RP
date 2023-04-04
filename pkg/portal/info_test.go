package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestInfo(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	defer controller.Finish()

	_env := mock_env.NewMockCore(controller)
	_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)
	_env.EXPECT().Location().AnyTimes().Return("eastus")
	_env.EXPECT().TenantID().AnyTimes().Return("00000000-0000-0000-0000-000000000001")
	_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
	_env.EXPECT().Hostname().AnyTimes().Return("testhost")

	dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()
	dbPortal, _ := testdatabase.NewFakePortal()

	p := NewTestPortal(_env, dbOpenShiftClusters, dbPortal)
	defer p.Cleanup()
	err := p.Run(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	for _, tt := range []struct {
		name               string
		expectedResponse   PortalInfo
		expectedStatusCode int
		authenticated      bool
		elevated           bool
	}{
		{
			name:               "basic",
			authenticated:      true,
			elevated:           false,
			expectedStatusCode: 200,
			expectedResponse: PortalInfo{
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
			expectedResponse: PortalInfo{
				Location:  "eastus",
				Username:  "username",
				Elevated:  true,
				RPVersion: version.GitCommit,
			},
		},
	} {
		resp, err := p.Request("GET", "/api/info", tt.authenticated, tt.elevated)
		if err != nil {
			p.DumpLogs(t)
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != tt.expectedStatusCode {
			t.Errorf("%d != %d", resp.StatusCode, tt.expectedStatusCode)
		}

		if resp.Header.Get("Content-Type") != "application/json" {
			t.Error(resp.Header.Get("Content-Type"))
		}

		var readResp PortalInfo
		err = json.NewDecoder(resp.Body).Decode(&readResp)
		if err != nil {
			t.Fatal(err)
		}

		// copy through the CSRF token if it's non-blank, since we can't make it
		// a known value
		if readResp.CSRFToken != "" {
			tt.expectedResponse.CSRFToken = readResp.CSRFToken
		}

		for _, l := range deep.Equal(readResp, tt.expectedResponse) {
			t.Error(l)
		}
	}
}
