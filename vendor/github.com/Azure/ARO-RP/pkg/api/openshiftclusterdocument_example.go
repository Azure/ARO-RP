package api

import "time"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ExampleOpenShiftClusterDocument() *OpenShiftClusterDocument {
	timestampString := "2020-02-03T01:01:01.1075056Z"
	timestamp, err := time.Parse(time.RFC3339, timestampString)
	if err != nil {
		panic(err)
	}

	return &OpenShiftClusterDocument{
		ID:                        "00000000-0000-0000-0000-000000000000",
		Key:                       "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
		Bucket:                    42,
		ClusterResourceGroupIDKey: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterresourcegroup",
		ClientIDKey:               "11111111-1111-1111-1111-111111111111",
		OpenShiftCluster: &OpenShiftCluster{
			ID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName",
			Name:     "resourceName",
			Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
			Location: "location",
			SystemData: SystemData{
				CreatedBy:          "string",
				CreatedByType:      CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedBy:     "string",
				LastModifiedByType: CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			Tags: map[string]string{
				"key": "value",
			},
			Identity: &ManagedServiceIdentity{
				Type: ManagedServiceIdentityUserAssigned,
				UserAssignedIdentities: map[string]UserAssignedIdentity{
					"": {},
				},
			},
			Properties: OpenShiftClusterProperties{
				ProvisioningState: ProvisioningStateSucceeded,
				ClusterProfile: ClusterProfile{
					PullSecret:      `{"auths":{"registry.connect.redhat.com":{"auth":""},"registry.redhat.io":{"auth":""}}}`,
					Domain:          "cluster.location.aroapp.io",
					Version:         "4.11.0",
					ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/clusterResourceGroup",
				},
				ConsoleProfile: ConsoleProfile{
					URL: "https://console-openshift-console.apps.cluster.location.aroapp.io/",
				},
				ServicePrincipalProfile: &ServicePrincipalProfile{
					ClientSecret: "clientSecret",
					ClientID:     "clientId",
				},
				NetworkProfile: NetworkProfile{
					PodCIDR:          "10.128.0.0/14",
					ServiceCIDR:      "172.30.0.0/16",
					PreconfiguredNSG: PreconfiguredNSGDisabled,
				},
				MasterProfile: MasterProfile{
					VMSize:   VMSizeStandardD8sV3,
					SubnetID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
				},
				WorkerProfiles: []WorkerProfile{
					{
						Name:       "worker",
						VMSize:     VMSizeStandardD2sV3,
						DiskSizeGB: 128,
						SubnetID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
						Count:      3,
					},
				},
				WorkerProfilesStatus: []WorkerProfile{
					{
						Name:       "worker1",
						VMSize:     VMSizeStandardD2sV3,
						DiskSizeGB: 128,
						SubnetID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
						Count:      1,
					},
					{
						Name:       "worker2",
						VMSize:     VMSizeStandardD2sV3,
						DiskSizeGB: 128,
						SubnetID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
						Count:      1,
					},
					{
						Name:       "worker3",
						VMSize:     VMSizeStandardD2sV3,
						DiskSizeGB: 128,
						SubnetID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
						Count:      1,
					},
				},
				APIServerProfile: APIServerProfile{
					Visibility: VisibilityPublic,
					URL:        "https://api.cluster.location.aroapp.io:6443/",
					IP:         "1.2.3.4",
				},
				IngressProfiles: []IngressProfile{
					{
						Name:       "default",
						Visibility: VisibilityPublic,
						IP:         "1.2.3.4",
					},
				},
				Install: &Install{
					Now:   time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
					Phase: InstallPhaseBootstrap,
				},
				StorageSuffix:     "rexs1",
				SSHKey:            SecureBytes("ssh-key"),
				AdminKubeconfig:   SecureBytes("admin-kubeconfig"),
				KubeadminPassword: SecureString("password"),
			},
		},
	}
}
