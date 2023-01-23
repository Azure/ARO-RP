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
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
)

func TestNewGateway(t *testing.T) {
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
			metrics := mock_metrics.NewMockEmitter(controller)

			httpl, _ := utilnet.Listen("tcp", ":8080", SocketSize)
			httpsl, _ := utilnet.Listen("tcp", ":8443", SocketSize)
			healthListener, _ := utilnet.Listen("tcp", ":8081", SocketSize)

			env := mock_env.NewMockCore(controller)
			tt.mocks(env)

			gtwy, err := NewGateway(ctx, env, baseLog, baseLog, nil, httpsl, httpl, healthListener, tt.acrResourceID, tt.gatewayDomains, metrics)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("Expected error: %q but did not receive any errors", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("The expected error: %q did not match the actual error: %q", tt.wantErr, err.Error())
				}
				return // no need to continue processing the tests
			} else if err != nil {
				t.Errorf("Got an unexpected error: %q", err.Error())
				return
			}

			gtwyValue, _ := gtwy.(*gateway)

			if len(gtwyValue.allowList) != len(tt.expectedAllowList) {
				t.Errorf("expected %d but got %d domains in the allow list", len(tt.expectedAllowList), len(gtwyValue.allowList))
			}

			for _, expectedDomain := range tt.expectedAllowList {
				_, ok := gtwyValue.allowList[expectedDomain]
				if !ok {
					t.Errorf("wanted domain: %q but it was missing", expectedDomain)
				}
			}
		})
	}
}

func TestNewGatewayDefaultConditions(t *testing.T) {
	populatedEnv := &azureclient.AROEnvironment{
		Environment: azure.Environment{
			ActiveDirectoryEndpoint:    "https://activedirectory/redhattest",
			ResourceManagerEndpoint:    "https://resourcemanager/redhattest",
			ContainerRegistryDNSSuffix: "dnsSuffix",
		},
	}

	httpl, _ := utilnet.Listen("tcp", ":8080", SocketSize)
	httpsl, _ := utilnet.Listen("tcp", ":8443", SocketSize)
	healthListener, _ := utilnet.Listen("tcp", ":8081", SocketSize)
	acrResourceID := "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/resourceName"
	gatewayDomains := "doMain1,Domain2"
	ctx := context.Background()
	baseLog := logrus.NewEntry(logrus.StandardLogger())

	controller := gomock.NewController(t)
	defer controller.Finish()
	metrics := mock_metrics.NewMockEmitter(controller)
	env := mock_env.NewMockCore(controller)
	env.EXPECT().Environment().AnyTimes().Return(populatedEnv)
	env.EXPECT().Location().AnyTimes().Return("location")

	gtwy, _ := NewGateway(ctx, env, baseLog, baseLog, nil, httpsl, httpl, healthListener, acrResourceID, gatewayDomains, metrics)

	gateway, _ := gtwy.(*gateway)

	for _, tt := range []struct {
		name                           string
		failedGatewayCreationCondition bool
		failedGatewayCreationReason    string
	}{
		{
			name:                           "httpl is set",
			failedGatewayCreationCondition: gateway.httpl.(*proxyproto.Listener).Listener != httpl,
			failedGatewayCreationReason:    "httpl was not set properly",
		},
		{
			name:                           "httpsl is set",
			failedGatewayCreationCondition: gateway.httpsl.(*proxyproto.Listener).Listener != httpsl,
			failedGatewayCreationReason:    "httpsl was not set properly",
		},
		{
			name:                           "Read timeout is set",
			failedGatewayCreationCondition: gateway.server.ReadTimeout != 10*time.Second,
			failedGatewayCreationReason:    "gateway http server timeout should be set to 10s",
		},
		{
			name:                           "Idle timeout is set",
			failedGatewayCreationCondition: gateway.server.IdleTimeout != 2*time.Minute,
			failedGatewayCreationReason:    "gateway http server timeout should be set to 2m0s",
		},
		{
			name:                           "error log prefix is set",
			failedGatewayCreationCondition: gateway.server.ErrorLog.Prefix() != "",
			failedGatewayCreationReason:    "gateway error log prefix should be blank",
		},
		{
			name:                           "error log flags are set to 0",
			failedGatewayCreationCondition: gateway.server.ErrorLog.Flags() != 0,
			failedGatewayCreationReason:    "gateway error log flags should be set to 0",
		},
		{
			name:                           "baseContext is the context of the gateway",
			failedGatewayCreationCondition: gateway.server.BaseContext(httpl) != ctx,
			failedGatewayCreationReason:    "gateway http server BaseContext should return the ctx of the gateway",
		},
		{
			name:                           "gateway http server handler is set",
			failedGatewayCreationCondition: gateway.server.Handler == nil,
			failedGatewayCreationReason:    "gateway http server should have its handler set",
		},
		{
			name:                           "a new gateway has an empty list of gateways",
			failedGatewayCreationCondition: len(gateway.gateways) != 0,
			failedGatewayCreationReason:    "gateways should be empty",
		},
		{
			name:                           "gateways metrics is set",
			failedGatewayCreationCondition: gateway.m != metrics,
			failedGatewayCreationReason:    "gateway metrics are not set properly",
		},
		{
			name:                           "gateway is ready on creation",
			failedGatewayCreationCondition: gateway.ready.Load() != true,
			failedGatewayCreationReason:    "gateway is not ready when it should be",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.failedGatewayCreationCondition {
				t.Error(tt.failedGatewayCreationReason)
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
