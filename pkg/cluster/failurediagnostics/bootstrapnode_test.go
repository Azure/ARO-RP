package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	bsTestResourceGroup   = "resourceGroupCluster"
	bsTestResourceGroupID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/" + bsTestResourceGroup
	bsTestInfraID         = "infra"
	bsTestLBID            = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/" + bsTestResourceGroup + "/providers/Microsoft.Network/loadBalancers/" + bsTestInfraID + "-internal"
)

func newBSTestDoc(resourceGroupID, infraID string) *api.OpenShiftClusterDocument {
	return &api.OpenShiftClusterDocument{
		Key: "testkey",
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				InfraID: infraID,
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: resourceGroupID,
				},
				NetworkProfile: api.NetworkProfile{
					APIServerPrivateEndpointIP: "10.0.0.1",
				},
			},
		},
	}
}

func TestLogBootstrapNode(t *testing.T) {
	for _, tt := range []struct {
		name       string
		doc        *api.OpenShiftClusterDocument
		wantOutput []any
	}{
		{
			name:       "nil clients returns descriptive entry without panic",
			doc:        newBSTestDoc(bsTestResourceGroupID, bsTestInfraID),
			wantOutput: []any{"lb or interface client missing"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, log := testlog.New()

			m := &manager{
				log: log,
				doc: tt.doc,
			}

			out, err := m.LogBootstrapNode(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantOutput != nil {
				for _, d := range deep.Equal(out, tt.wantOutput) {
					t.Error(d)
				}
			}
		})
	}
}

func TestSetupBootstrapNodeSSH(t *testing.T) {
	for _, tt := range []struct {
		name            string
		doc             *api.OpenShiftClusterDocument
		mockLB          func(*mock_armnetwork.MockLoadBalancersClient)
		mockNIC         func(*mock_armnetwork.MockInterfacesClient)
		wantErrContains string
	}{
		{
			name:            "empty InfraID returns error",
			doc:             newBSTestDoc(bsTestResourceGroupID, ""),
			wantErrContains: "infraID is not set",
		},
		{
			name: "empty APIServerPrivateEndpointIP returns error",
			doc: func() *api.OpenShiftClusterDocument {
				d := newBSTestDoc(bsTestResourceGroupID, bsTestInfraID)
				d.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP = ""
				return d
			}(),
			wantErrContains: "APIServerPrivateEndpointIP is not set",
		},
		{
			name: "LB Get failure returns error",
			doc:  newBSTestDoc(bsTestResourceGroupID, bsTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().
					Get(gomock.Any(), bsTestResourceGroup, bsTestInfraID+"-internal", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{}, errors.New("lb get failed"))
			},
			mockNIC:         func(*mock_armnetwork.MockInterfacesClient) {},
			wantErrContains: "lb get failed",
		},
		{
			name: "NIC Get failure returns error",
			doc:  newBSTestDoc(bsTestResourceGroupID, bsTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				lb := makeTestLBWithBootstrapConfig()
				m.EXPECT().
					Get(gomock.Any(), bsTestResourceGroup, bsTestInfraID+"-internal", nil).
					Return(armnetwork.LoadBalancersClientGetResponse{LoadBalancer: lb}, nil)
			},
			mockNIC: func(m *mock_armnetwork.MockInterfacesClient) {
				m.EXPECT().
					Get(gomock.Any(), bsTestResourceGroup, bsTestInfraID+"-bootstrap-nic", nil).
					Return(armnetwork.InterfacesClientGetResponse{}, errors.New("nic get failed"))
			},
			wantErrContains: "nic get failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, log := testlog.New()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := &manager{
				log: log,
				doc: tt.doc,
			}

			if tt.mockLB != nil {
				lbClient := mock_armnetwork.NewMockLoadBalancersClient(ctrl)
				tt.mockLB(lbClient)
				m.loadBalancers = lbClient
			}
			if tt.mockNIC != nil {
				nicClient := mock_armnetwork.NewMockInterfacesClient(ctrl)
				tt.mockNIC(nicClient)
				m.armInterfaces = nicClient
			}

			_, err := m.setupBootstrapNodeSSH(ctx)

			if tt.wantErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("want error containing %q, got %v", tt.wantErrContains, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnsureBootstrapLBConfig(t *testing.T) {
	for _, tt := range []struct {
		name            string
		lb              armnetwork.LoadBalancer
		wantChanged     bool
		wantErrContains string
		wantProbes      []string
		wantPools       []string
		wantRules       []string
	}{
		{
			name:        "empty LB gets all three resources added",
			lb:          makeMinimalLB(),
			wantChanged: true,
			wantProbes:  []string{bootstrapNodeSSHProbeName},
			wantPools:   []string{bootstrapNodeBackendPool},
			wantRules:   []string{bootstrapNodeBackendPool},
		},
		{
			name:        "LB already fully configured is not modified",
			lb:          makeTestLBWithBootstrapConfig(),
			wantChanged: false,
			wantProbes:  []string{bootstrapNodeSSHProbeName},
			wantPools:   []string{bootstrapNodeBackendPool},
			wantRules:   []string{bootstrapNodeBackendPool},
		},
		{
			name: "LB missing only the rule gets rule added",
			lb: func() armnetwork.LoadBalancer {
				lb := makeMinimalLB()
				lb.Properties.Probes = []*armnetwork.Probe{{Name: pointerutils.ToPtr(bootstrapNodeSSHProbeName)}}
				lb.Properties.BackendAddressPools = []*armnetwork.BackendAddressPool{{Name: pointerutils.ToPtr(bootstrapNodeBackendPool)}}
				return lb
			}(),
			wantChanged: true,
			wantProbes:  []string{bootstrapNodeSSHProbeName},
			wantPools:   []string{bootstrapNodeBackendPool},
			wantRules:   []string{bootstrapNodeBackendPool},
		},
		{
			name:            "nil properties returns error",
			lb:              armnetwork.LoadBalancer{},
			wantErrContains: "nil properties",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.New()
			m := &manager{log: log}

			got, err := m.ensureBootstrapNodeLBConfig(&tt.lb)

			if tt.wantErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("want error containing %q, got %v", tt.wantErrContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.wantChanged {
				t.Errorf("changed = %v, want %v", got, tt.wantChanged)
			}

			probeNames := namesFrom(lbProbes(tt.lb))
			for _, want := range tt.wantProbes {
				if !contains(probeNames, want) {
					t.Errorf("probe %q not found in %v", want, probeNames)
				}
			}
			poolNames := namesFrom(lbPools(tt.lb))
			for _, want := range tt.wantPools {
				if !contains(poolNames, want) {
					t.Errorf("pool %q not found in %v", want, poolNames)
				}
			}
			ruleNames := namesFrom(lbRules(tt.lb))
			for _, want := range tt.wantRules {
				if !contains(ruleNames, want) {
					t.Errorf("rule %q not found in %v", want, ruleNames)
				}
			}
		})
	}
}

func TestEnsureNICInBootstrapPool(t *testing.T) {
	poolID := bsTestLBID + "/backendAddressPools/" + bootstrapNodeBackendPool

	for _, tt := range []struct {
		name        string
		nic         armnetwork.Interface
		wantChanged bool
	}{
		{
			name:        "NIC with no pools gets pool added",
			nic:         makeTestNIC(nil),
			wantChanged: true,
		},
		{
			name:        "NIC already in pool is not modified",
			nic:         makeTestNIC([]string{poolID}),
			wantChanged: false,
		},
		{
			name:        "NIC in different pool gets bootstrap pool added",
			nic:         makeTestNIC([]string{bsTestLBID + "/backendAddressPools/other-pool"}),
			wantChanged: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureNICInBootstrapNodePool(&tt.nic, poolID)
			if got != tt.wantChanged {
				t.Errorf("changed = %v, want %v", got, tt.wantChanged)
			}
			if tt.wantChanged {
				found := false
				for _, ipc := range tt.nic.Properties.IPConfigurations {
					for _, p := range ipc.Properties.LoadBalancerBackendAddressPools {
						if p.ID != nil && strings.EqualFold(*p.ID, poolID) {
							found = true
						}
					}
				}
				if !found {
					t.Errorf("pool %q not found in NIC after update", poolID)
				}
			}
		})
	}
}

// ---- helpers ----

func makeMinimalLB() armnetwork.LoadBalancer {
	return armnetwork.LoadBalancer{
		ID: pointerutils.ToPtr(bsTestLBID),
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{ID: pointerutils.ToPtr(bsTestLBID + "/frontendIPConfigurations/public-lb-ip-v4")},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{},
			LoadBalancingRules:  []*armnetwork.LoadBalancingRule{},
			Probes:              []*armnetwork.Probe{},
		},
	}
}

func makeTestLBWithBootstrapConfig() armnetwork.LoadBalancer {
	lb := makeMinimalLB()
	lb.Properties.Probes = []*armnetwork.Probe{
		{Name: pointerutils.ToPtr(bootstrapNodeSSHProbeName)},
	}
	lb.Properties.BackendAddressPools = []*armnetwork.BackendAddressPool{
		{Name: pointerutils.ToPtr(bootstrapNodeBackendPool)},
	}
	lb.Properties.LoadBalancingRules = []*armnetwork.LoadBalancingRule{
		{Name: pointerutils.ToPtr(bootstrapNodeBackendPool)},
	}
	return lb
}

func makeTestNIC(poolIDs []string) armnetwork.Interface {
	pools := make([]*armnetwork.BackendAddressPool, 0, len(poolIDs))
	for _, id := range poolIDs {
		pools = append(pools, &armnetwork.BackendAddressPool{ID: &id})
	}
	return armnetwork.Interface{
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						LoadBalancerBackendAddressPools: pools,
					},
				},
			},
		},
	}
}

// Helpers to extract names from LB resource slices.

func lbProbes(lb armnetwork.LoadBalancer) []*string {
	out := make([]*string, 0, len(lb.Properties.Probes))
	for _, p := range lb.Properties.Probes {
		out = append(out, p.Name)
	}
	return out
}

func lbPools(lb armnetwork.LoadBalancer) []*string {
	out := make([]*string, 0, len(lb.Properties.BackendAddressPools))
	for _, p := range lb.Properties.BackendAddressPools {
		out = append(out, p.Name)
	}
	return out
}

func lbRules(lb armnetwork.LoadBalancer) []*string {
	out := make([]*string, 0, len(lb.Properties.LoadBalancingRules))
	for _, r := range lb.Properties.LoadBalancingRules {
		out = append(out, r.Name)
	}
	return out
}

func namesFrom(ptrs []*string) []string {
	out := make([]string, 0, len(ptrs))
	for _, p := range ptrs {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
