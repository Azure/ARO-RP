package loadbalancer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/ARO-RP/pkg/api"
	fakelb "github.com/Azure/ARO-RP/pkg/util/loadbalancer/fake"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	infraID                  = "infraID"
	location                 = "eastus"
	publicIngressFIPConfigID = to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/ae3506385907e44eba9ef9bf76eac973")
)

func TestRemoveLoadBalancerFrontendIPConfiguration(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name          string
		fipResourceID string
		originalLB    func() *mgmtnetwork.LoadBalancer
		expectedLB    func() *mgmtnetwork.LoadBalancer
		expectedErr   string
	}{
		{
			name:          "remove frontend ip config",
			fipResourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083",
			originalLB: func() *mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations, NewFrontendIPConfig("adce98f85c7dd47c5a21263a5e39c083", "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/adce98f85c7dd47c5a21263a5e39c083", "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-adce98f85c7dd47c5a21263a5e39c083"))
				return &lb
			},
			expectedLB: func() *mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				return &lb
			},
		},
		{
			name:          "removal of frontend ip config fails when frontend ip config has references",
			fipResourceID: *publicIngressFIPConfigID,
			originalLB: func() *mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				return &lb
			},
			expectedLB: func() *mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				return &lb
			},
			expectedErr: fmt.Sprintf("frontend IP Configuration %s has external references, remove the external references prior to removing the frontend IP configuration", *publicIngressFIPConfigID),
		},
		{
			name:          "removal of frontend ip config fails when frontend ip config doesn't exist",
			fipResourceID: *publicIngressFIPConfigID,
			originalLB: func() *mgmtnetwork.LoadBalancer {
				return &mgmtnetwork.LoadBalancer{
					LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{},
				}
			},
			expectedLB: func() *mgmtnetwork.LoadBalancer {
				return &mgmtnetwork.LoadBalancer{
					LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{},
				}
			},
			expectedErr: "FrontendIPConfigurations in nil",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			originalLB := tt.originalLB()
			err := RemoveFrontendIPConfiguration(originalLB, tt.fipResourceID)
			assert.Equal(t, tt.expectedLB(), originalLB)
			utilerror.AssertErrorMessage(t, err, tt.expectedErr)
		})
	}
}

func TestGetOutboundIPsFromLB(t *testing.T) {
	// Run tests
	for _, tt := range []struct {
		name                string
		originalLB          mgmtnetwork.LoadBalancer
		expectedOutboundIPs []api.ResourceReference
	}{
		{
			name:       "default",
			originalLB: fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic),
			expectedOutboundIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run GetOutboundIPsFromLB and assert the correct results
			outboundIPs := GetOutboundIPsFromLB(tt.originalLB)
			assert.Equal(t, tt.expectedOutboundIPs, outboundIPs)
		})
	}
}

func TestAddOutboundIPsToLB(t *testing.T) {
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"

	// Run tests
	for _, tt := range []struct {
		name         string
		desiredOBIPs []api.ResourceReference
		originalLB   func() mgmtnetwork.LoadBalancer
		expectedLB   func() mgmtnetwork.LoadBalancer
	}{
		{
			name: "add default IP to lb",
			desiredOBIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
			},
			originalLB: func() mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations = &[]mgmtnetwork.SubResource{}
				return lb
			},
			expectedLB: func() mgmtnetwork.LoadBalancer {
				return fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
			},
		},
		{
			name: "add multiple outbound IPs to LB",
			desiredOBIPs: []api.ResourceReference{
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/infraID-pip-v4",
				},
				{
					ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
				},
			},
			originalLB: func() mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations = &[]mgmtnetwork.SubResource{}
				return lb
			},
			expectedLB: func() mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations,
					NewFrontendIPConfig(
						"uuid1-outbound-pip-v4",
						fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", clusterRGID, infraID, "uuid1-outbound-pip-v4"),
						fmt.Sprintf("%s/providers/Microsoft.Network/publicIPAddresses/%s", clusterRGID, "uuid1-outbound-pip-v4"),
					),
				)
				*(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations = append(*(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations, NewOutboundRuleFrontendIPConfig(fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", clusterRGID, infraID, "uuid1-outbound-pip-v4")))
				return lb
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Run addOutboundIPsToLB and assert the correct results
			originalLB := tt.originalLB()
			AddOutboundIPsToLB(clusterRGID, originalLB, tt.desiredOBIPs)
			assert.Equal(t, tt.expectedLB(), originalLB)
		})
	}
}

func TestRemoveOutboundIPsFromLB(t *testing.T) {
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"

	// Run tests
	for _, tt := range []struct {
		name       string
		originalLB func() mgmtnetwork.LoadBalancer
		expectedLB func() mgmtnetwork.LoadBalancer
	}{
		{
			name: "remove all outbound-rule-v4 fip config except api server",
			originalLB: func() mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)
				*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations,
					NewFrontendIPConfig(
						"uuid1-outbound-pip-v4",
						fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/FrontendIPConfigurations/%s", clusterRGID, infraID, "uuid1-outbound-pip-v4"),
						fmt.Sprintf("%s/providers/Microsoft.Network/publicIPAddresses/%s", clusterRGID, "uuid1-outbound-pip-v4"),
					),
				)
				*(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations = append(*(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations, NewOutboundRuleFrontendIPConfig(fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/FrontendIPConfigurations/%s", clusterRGID, infraID, "uuid1-outbound-pip-v4")))
				return lb
			},
			expectedLB: func() mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPublic)

				(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations = &[]mgmtnetwork.SubResource{}
				return lb
			},
		},
		{
			name: "remove all outbound-rule-v4 fip config",
			originalLB: func() mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPrivate)
				*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = append(*lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations,
					NewFrontendIPConfig(
						"uuid1-outbound-pip-v4",
						fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/FrontendIPConfigurations/%s", clusterRGID, infraID, "uuid1-outbound-pip-v4"),
						fmt.Sprintf("%s/providers/Microsoft.Network/publicIPAddresses/%s", clusterRGID, "uuid1-outbound-pip-v4"),
					),
				)
				*(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations = append(*(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations, NewOutboundRuleFrontendIPConfig(fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/FrontendIPConfigurations/%s", clusterRGID, infraID, "uuid1-outbound-pip-v4")))
				return lb
			},
			expectedLB: func() mgmtnetwork.LoadBalancer {
				lb := fakelb.NewFakePublicLoadBalancer(api.VisibilityPrivate)
				lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations = &[]mgmtnetwork.FrontendIPConfiguration{
					fakelb.FakeDefaultIngressFrontendIPConfig,
				}
				(*lb.LoadBalancerPropertiesFormat.OutboundRules)[0].FrontendIPConfigurations = &[]mgmtnetwork.SubResource{}
				return lb
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			originalLB := tt.originalLB()
			expectedLB := tt.expectedLB()

			// Run RemoveOutboundIPsFromLB and assert correct results
			RemoveOutboundIPsFromLB(originalLB)
			assert.Equal(t, expectedLB, originalLB)
		})
	}
}
