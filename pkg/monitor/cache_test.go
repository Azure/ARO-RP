package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestStripUnusedFields(t *testing.T) {
	tests := []struct {
		name     string
		input    *api.OpenShiftClusterDocument
		validate func(*testing.T, *api.OpenShiftClusterDocument)
	}{
		{
			name:  "nil document returns nil",
			input: nil,
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if result != nil {
					t.Error("expected nil result for nil input")
				}
			},
		},
		{
			name: "nil OpenShiftCluster returns original",
			input: &api.OpenShiftClusterDocument{
				ID:               "test-id",
				OpenShiftCluster: nil,
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if result == nil || result.ID != "test-id" {
					t.Error("expected original document with nil OpenShiftCluster")
				}
			},
		},
		{
			name: "strips sensitive fields",
			input: &api.OpenShiftClusterDocument{
				ID:           "cluster-1",
				Key:          "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-1",
				PartitionKey: "partition-1",
				Bucket:       5,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-1",
					Name:     "cluster-1",
					Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
					Location: "eastus",
					Properties: api.OpenShiftClusterProperties{
						ProvisioningState: api.ProvisioningStateSucceeded,
						ClusterProfile: api.ClusterProfile{
							PullSecret: "super-secret-pull-secret",
							Domain:     "test.example.com",
							Version:    "4.12.0",
						},
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "10.0.0.1",
							PreconfiguredNSG:           api.PreconfiguredNSGDisabled,
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet1/subnets/master",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								Name:     "worker",
								SubnetID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet1/subnets/worker",
								Count:    3,
								VMSize:   "Standard_D4s_v3",
							},
						},
						APIServerProfile: api.APIServerProfile{
							URL: "https://api.test.example.com:6443",
						},
						SSHKey:            api.SecureBytes("ssh-rsa AAAAB3..."),
						AdminKubeconfig:   api.SecureBytes("admin-kubeconfig-data"),
						KubeadminPassword: "admin-password",
						RegistryProfiles: []*api.RegistryProfile{
							{
								Name:     "registry1",
								Username: "user1",
								Password: "password1",
							},
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							ClientID:     "client-id-123",
							ClientSecret: "super-secret-client-secret",
						},
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				// Verify document metadata preserved
				if result.ID != "cluster-1" {
					t.Errorf("expected ID 'cluster-1', got %s", result.ID)
				}
				if result.Key == "" {
					t.Error("expected Key to be preserved")
				}
				if result.Bucket != 5 {
					t.Errorf("expected Bucket 5, got %d", result.Bucket)
				}

				// Verify cluster identity preserved
				oc := result.OpenShiftCluster
				if oc.ID == "" || oc.Name != "cluster-1" || oc.Location != "eastus" {
					t.Error("cluster identity fields should be preserved")
				}

				// Verify essential properties preserved
				props := oc.Properties
				if props.ProvisioningState != api.ProvisioningStateSucceeded {
					t.Error("ProvisioningState should be preserved")
				}
				if props.NetworkProfile.APIServerPrivateEndpointIP != "10.0.0.1" {
					t.Error("APIServerPrivateEndpointIP should be preserved")
				}
				if props.MasterProfile.SubnetID == "" {
					t.Error("MasterProfile.SubnetID should be preserved")
				}
				if props.APIServerProfile.URL == "" {
					t.Error("APIServerProfile.URL should be preserved")
				}

				// Verify sensitive fields stripped
				if props.ClusterProfile.PullSecret != "" {
					t.Error("PullSecret should be stripped")
				}
				if props.SSHKey != nil {
					t.Error("SSHKey should be stripped")
				}
				if props.KubeadminPassword != "" {
					t.Error("KubeadminPassword should be stripped")
				}
				if props.RegistryProfiles != nil {
					t.Error("RegistryProfiles should be stripped")
				}

				// Verify ServicePrincipalProfile has no secret
				if props.ServicePrincipalProfile == nil {
					t.Error("ServicePrincipalProfile presence should be preserved")
				} else if props.ServicePrincipalProfile.ClientSecret != "" {
					t.Error("ServicePrincipalProfile.ClientSecret should be stripped")
				} else if props.ServicePrincipalProfile.ClientID != "client-id-123" {
					t.Error("ServicePrincipalProfile.ClientID should be preserved")
				}

				// Verify worker profiles stripped of unnecessary fields
				if len(props.WorkerProfiles) != 1 {
					t.Error("WorkerProfiles should be preserved")
				} else {
					wp := props.WorkerProfiles[0]
					if wp.SubnetID == "" {
						t.Error("WorkerProfile.SubnetID should be preserved")
					}
					if wp.VMSize != "" {
						t.Error("WorkerProfile.VMSize should be stripped")
					}
				}
			},
		},
		{
			name: "prefers AROServiceKubeconfig over AdminKubeconfig",
			input: &api.OpenShiftClusterDocument{
				ID: "cluster-2",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-2",
					Name:     "cluster-2",
					Location: "westus",
					Properties: api.OpenShiftClusterProperties{
						AdminKubeconfig:      api.SecureBytes("admin-kubeconfig"),
						AROServiceKubeconfig: api.SecureBytes("aro-service-kubeconfig"),
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if string(result.OpenShiftCluster.Properties.AROServiceKubeconfig) != "aro-service-kubeconfig" {
					t.Error("should prefer AROServiceKubeconfig")
				}
				// AdminKubeconfig should not be separately stored
				if result.OpenShiftCluster.Properties.AdminKubeconfig != nil {
					t.Error("AdminKubeconfig should not be stored when AROServiceKubeconfig exists")
				}
			},
		},
		{
			name: "uses AdminKubeconfig when AROServiceKubeconfig is nil",
			input: &api.OpenShiftClusterDocument{
				ID: "cluster-3",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-3",
					Name:     "cluster-3",
					Location: "eastus2",
					Properties: api.OpenShiftClusterProperties{
						AdminKubeconfig:      api.SecureBytes("admin-kubeconfig-only"),
						AROServiceKubeconfig: nil,
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if string(result.OpenShiftCluster.Properties.AROServiceKubeconfig) != "admin-kubeconfig-only" {
					t.Error("should use AdminKubeconfig when AROServiceKubeconfig is nil")
				}
			},
		},
		{
			name: "preserves PlatformWorkloadIdentityProfile presence",
			input: &api.OpenShiftClusterDocument{
				ID: "cluster-4",
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:       "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster-4",
					Name:     "cluster-4",
					Location: "northeurope",
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"identity1": {
									ResourceID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id1",
									ClientID:   "client-1",
									ObjectID:   "object-1",
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *api.OpenShiftClusterDocument) {
				if result.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile == nil {
					t.Error("PlatformWorkloadIdentityProfile presence should be preserved")
				}
				// But the actual identities should be stripped
				if result.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities != nil {
					t.Error("PlatformWorkloadIdentities details should be stripped")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripUnusedFields(tt.input)
			tt.validate(t, result)
		})
	}
}
