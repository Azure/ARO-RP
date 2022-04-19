package gateway

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"
	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
)

func TestNewGatway(t *testing.T) {
	emptyEnv := &azureclient.AROEnvironment{
		Environment: azure.Environment{
			ActiveDirectoryEndpoint:    "",
			ResourceManagerEndpoint:    "",
			ContainerRegistryDNSSuffix: "dnsSuffix",
		},
	}

	populatedEnv := *emptyEnv
	populatedEnv.ActiveDirectoryEndpoint = "https://activedirectory/redhattest"
	populatedEnv.ResourceManagerEndpoint = "https://resourcemanager/redhattest"

	for _, tt := range []struct {
		name              string
		gatewayDomains    string
		acrResourceID     string
		mocks             func(env *mock_env.MockCore)
		expectedAllowList []string
		wantErr           string
	}{
		{
			name: "An environment without the required domains returns an error on creation",
			mocks: func(env *mock_env.MockCore) {
				env.EXPECT().Environment().AnyTimes().Return(emptyEnv)
			},
			expectedAllowList: make([]string, 0),
			wantErr:           "missing required domain. Ensure the environment has both ActiveDirectoryEndpoint and ResourceManagerEndpoint",
		},
		{
			name:           "GatewayDomains are split, formatted, and added to allowList",
			gatewayDomains: "doMain1,Domain2",
			mocks: func(env *mock_env.MockCore) {
				env.EXPECT().Environment().AnyTimes().Return(&populatedEnv)
			},
			expectedAllowList: []string{"domain1", "domain2", "activedirectory", "resourcemanager"},
		},
		{
			name: "Domains in Environments are parsed and appended to allowList",
			mocks: func(env *mock_env.MockCore) {
				env.EXPECT().Environment().AnyTimes().AnyTimes().Return(&populatedEnv)
			},
			expectedAllowList: []string{"activedirectory", "resourcemanager"},
		},
		{
			name: "Invalid domain format in environment",
			mocks: func(env *mock_env.MockCore) {
				invalidEnv := *emptyEnv
				invalidEnv.ActiveDirectoryEndpoint = "\r"
				env.EXPECT().Environment().AnyTimes().AnyTimes().Return(&invalidEnv)
			},
			wantErr: `parse "\r": net/url: invalid control character in URL`,
		},
		{
			name:          "When the acrResourceID is present it is parsed into domains and added to allowList",
			acrResourceID: "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/resourceName",
			mocks: func(env *mock_env.MockCore) {
				env.EXPECT().Environment().AnyTimes().Return(&populatedEnv)
				env.EXPECT().Location().AnyTimes().Return("location")
			},
			expectedAllowList: []string{"resourcename.dnssuffix", "resourcename.location.data.dnssuffix", "activedirectory", "resourcemanager"},
		},
		{
			name:          "When the acrResourceID is present but it is invalid",
			acrResourceID: "invalidAcrResourceID",
			mocks: func(env *mock_env.MockCore) {
				env.EXPECT().Environment().AnyTimes().Return(&populatedEnv)
				env.EXPECT().Location().AnyTimes().Return("location")
			},
			wantErr: "parsing failed for invalidAcrResourceID. Invalid resource Id format",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			baseLog := logrus.NewEntry(logrus.StandardLogger())
			db := mock_database.NewMockGateway(controller)
			metrics := mock_metrics.NewMockEmitter(controller)

			httpl, _ := utilnet.Listen("tcp", ":8080", SocketSize)
			httpsl, _ := utilnet.Listen("tcp", ":8443", SocketSize)

			env := mock_env.NewMockCore(controller)
			tt.mocks(env)

			gtwy, err := NewGateway(ctx, env, baseLog, baseLog, db, httpsl, httpl, tt.acrResourceID, tt.gatewayDomains, metrics)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("Expected error: '%s' but did not receive any errors", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("The expected error: '%s' did not match the actual error: '%s'", tt.wantErr, err.Error())
				}
				return // no need to continue processing the tests
			} else if err != nil {
				t.Errorf("Got an unexpected error: %s", err.Error())
				return
			}

			gtwyValue, _ := gtwy.(*gateway)

			if len(gtwyValue.allowList) != len(tt.expectedAllowList) {
				t.Errorf("expected %d but got %d domains in the allow list", len(tt.expectedAllowList), len(gtwyValue.allowList))
			}
			for _, expectedDomain := range tt.expectedAllowList {
				_, ok := gtwyValue.allowList[expectedDomain]
				if !ok {
					t.Errorf("wanted domain: %s but it was missing", expectedDomain)
				}
			}

			if gtwyValue.httpl.(*proxyproto.Listener).Listener != httpl {
				t.Error("httpl was not set properly")
			}
			if gtwyValue.httpsl.(*proxyproto.Listener).Listener != httpsl {
				t.Error("httpsl was not set properly")
			}
			if gtwyValue.s.ReadTimeout != 10*time.Second {
				t.Errorf("gateway http server timeout set to %v when it should be 10s", gtwyValue.s.ReadTimeout)
			}
			if gtwyValue.s.IdleTimeout != 2*time.Minute {
				t.Errorf("gateway http server timeout set to %v when it should be 2m0s", gtwyValue.s.IdleTimeout)
			}
			if gtwyValue.s.ErrorLog.Prefix() != "" {
				t.Errorf("gateway http server ErrorLog should have empty prefix but was found with: %s", gtwyValue.s.ErrorLog.Prefix())
			}
			if gtwyValue.s.ErrorLog.Flags() != 0 {
				t.Errorf("gateway http server ErrorLog should have flags set as 0 but was found with: %d", gtwyValue.s.ErrorLog.Flags())
			}
			if gtwyValue.s.BaseContext(httpl) != ctx {
				t.Error("gateway http server BaseContext should return the ctx of the gateway")
			}
			if gtwyValue.s.Handler == nil {
				t.Error("gateway http server should have its handler set")
			}
			if len(gtwyValue.gateways) != 0 {
				t.Error("gateways should initially be empty")
			}
			if gtwyValue.m != metrics {
				t.Error("metrics was not set properly")
			}
			if gtwyValue.ready.Load() != true {
				t.Error("a new gateway should start as ready")
			}
		})
	}
}

func TestUpdateGateway(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		gatewaysAlreadyPresent []*api.Gateway
		doc                    *api.GatewayDocument
		wantValue              bool
		isDeleting             bool
	}{
		{
			name:      "add document",
			doc:       &api.GatewayDocument{ID: "notDeleting", Gateway: &api.Gateway{}},
			wantValue: true,
		},
		{
			name: "doesn't add document",
			doc: &api.GatewayDocument{ID: "deleting", Gateway: &api.Gateway{
				Deleting: true,
			}},
			wantValue: false,
		},
		{
			name:                   "remove document",
			gatewaysAlreadyPresent: []*api.Gateway{{ID: "toDelete"}},
			doc: &api.GatewayDocument{ID: "toDelete", Gateway: &api.Gateway{
				Deleting: true,
			}},
			wantValue: false,
		},
	} {
		gateway := gateway{
			gateways: make(map[string]*api.Gateway),
		}

		for _, v := range tt.gatewaysAlreadyPresent {
			gateway.gateways[v.ID] = v
		}

		t.Run(tt.name, func(t *testing.T) {
			gateway.updateGateways([]*api.GatewayDocument{tt.doc})

			if _, ok := gateway.gateways[tt.doc.ID]; ok != tt.wantValue {
				t.Error(tt.doc.ID)
			}
			if doc := gateway.gateways[tt.doc.ID]; tt.wantValue && doc == nil {
				t.Error(doc)
			}
		})
	}
}
