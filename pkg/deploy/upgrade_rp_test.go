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

func TestRPLoadBalancerRulesHealthy(t *testing.T) {
	ctx := context.Background()
	vmssName := rpVMSSPrefix + "test"
	resourceGroup := "rp-rg"
	instanceIDs := []string{"0", "1"}
	ruleNames := []string{rpRuleName, portalRuleName, portalSSHRuleName}

	backend := func(instanceID, state string) *armnetwork.LoadBalancerHealthPerRulePerBackendAddress {
		return &armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
			NetworkInterfaceIPConfigurationID: &armnetwork.InterfaceIPConfiguration{
				ID: pointerutils.ToPtr("/subscriptions/sub/resourceGroups/rp-rg/providers/Microsoft.Compute/virtualMachineScaleSets/" + vmssName + "/virtualMachines/" + instanceID + "/networkInterfaces/nic/ipConfigurations/ipconfig"),
			},
			State: pointerutils.ToPtr(state),
		}
	}

	for _, tt := range []struct {
		name        string
		health      map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse
		healthError map[string]error
		rules       []string
		wantReady   bool
		wantError   bool
	}{
		{
			name: "all instances healthy on every rule",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				rpRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"), backend("1", "Up"),
						},
					},
				},
				portalRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"), backend("1", "Up"),
						},
					},
				},
				portalSSHRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"), backend("1", "Up"),
						},
					},
				},
			},
			rules:     ruleNames,
			wantReady: true,
		},
		{
			name: "instance unhealthy on a rule",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				rpRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"), backend("1", "Down"),
						},
					},
				},
			},
			rules: []string{rpRuleName},
		},
		{
			name: "new instance missing from a rule",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				rpRuleName: {
					LoadBalancerHealthPerRule: armnetwork.LoadBalancerHealthPerRule{
						LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerHealthPerRulePerBackendAddress{
							backend("0", "Up"),
						},
					},
				},
			},
			rules: []string{rpRuleName},
		},
		{
			name: "load balancer health request fails",
			health: map[string]armnetwork.LoadBalancerLoadBalancingRulesClientHealthResponse{
				rpRuleName: {},
			},
			healthError: map[string]error{
				rpRuleName: errors.New("network failure"),
			},
			rules:     []string{rpRuleName},
			wantError: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			rulesClient := mock_armnetwork.NewMockLoadBalancerLoadBalancingRulesClient(controller)
			for _, ruleName := range tt.rules {
				rulesClient.EXPECT().
					HealthAndWait(ctx, resourceGroup, rpLBName, ruleName, nil).
					Return(tt.health[ruleName], tt.healthError[ruleName])
			}

			d := &deployer{
				config:             &RPConfig{RPResourceGroupName: resourceGroup},
				loadbalancingrules: rulesClient,
			}

			ready, err := d.rpLoadBalancerRulesHealthy(ctx, vmssName, instanceIDs)
			if (err != nil) != tt.wantError {
				t.Fatalf("rpLoadBalancerRulesHealthy() error = %v, wantError %v", err, tt.wantError)
			}
			if ready != tt.wantReady {
				t.Fatalf("rpLoadBalancerRulesHealthy() = %v, want %v", ready, tt.wantReady)
			}
		})
	}
}
