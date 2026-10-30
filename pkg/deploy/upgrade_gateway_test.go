package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func TestGatewayLoadBalancerRulesHealthy(t *testing.T) {
	ctx := context.Background()
	vmssName := gatewayVMSSPrefix + "test"
	resourceGroup := "gateway-rg"
	instanceIDs := []string{"0", "1"}

	backend := func(instanceID, state string) *armnetwork.LoadBalancerHealthPerRulePerBackendAddress {
		return &armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
			NetworkInterfaceIPConfigurationID: &armnetwork.InterfaceIPConfiguration{
				ID: pointerutils.ToPtr("/subscriptions/sub/resourceGroups/gateway-rg/providers/Microsoft.Compute/virtualMachineScaleSets/" + vmssName + "/virtualMachines/" + instanceID + "/networkInterfaces/nic/ipConfigurations/ipconfig"),
			},
			State: pointerutils.ToPtr(state),
		}
	}

	for _, tt := range []struct {
		name              string
		health            map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse
		healthError       map[string]error
		expectedRuleNames []string
		wantReady         bool
		wantError         bool
	}{
		{
			name: "all instances healthy on every rule",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				gatewayHTTPRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"),
							backend("1", "Up"),
						},
					},
				},
				gatewayHTTPSRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"),
							backend("1", "Up"),
						},
					},
				},
				gatewayOTelRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"),
							backend("1", "Up"),
						},
					},
				},
			},
			expectedRuleNames: []string{gatewayHTTPRuleName, gatewayHTTPSRuleName, gatewayOTelRuleName},
			wantReady:         true,
		},
		{
			name: "one instance unhealthy on a probe",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				gatewayHTTPRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"),
							backend("1", "Down"),
						},
					},
				},
			},
			expectedRuleNames: []string{gatewayHTTPRuleName},
		},
		{
			name: "new instance missing from a probe",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				gatewayHTTPRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"),
						},
					},
				},
			},
			expectedRuleNames: []string{gatewayHTTPRuleName},
		},
		{
			name: "load balancer health request fails",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				gatewayHTTPRuleName: {},
			},
			healthError: map[string]error{
				gatewayHTTPRuleName: errors.New("network failure"),
			},
			expectedRuleNames: []string{gatewayHTTPRuleName},
			wantError:         true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			rulesClient := mock_armnetwork.NewMockLoadBalancerLoadBalancingRulesClient(controller)
			for _, ruleName := range tt.expectedRuleNames {
				rulesClient.EXPECT().
					HealthAndWait(ctx, resourceGroup, gatewayLBName, ruleName, nil).
					Return(tt.health[ruleName], tt.healthError[ruleName])
			}

			d := &deployer{
				config:             &RPConfig{GatewayResourceGroupName: resourceGroup},
				loadbalancingrules: rulesClient,
			}

			ready, err := d.gatewayLoadBalancerRulesHealthy(ctx, vmssName, instanceIDs)
			if (err != nil) != tt.wantError {
				t.Fatalf("gatewayLoadBalancerRulesHealthy() error = %v, wantError %v", err, tt.wantError)
			}
			if ready != tt.wantReady {
				t.Fatalf("gatewayLoadBalancerRulesHealthy() = %v, want %v", ready, tt.wantReady)
			}
		})
	}
}
