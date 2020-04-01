package api

import "time"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ExampleOpenShiftClusterDocument() *OpenShiftClusterDocument {
	return &OpenShiftClusterDocument{
		ID:                        "00000000-0000-0000-0000-000000000000",
		Key:                       "/subscriptions/subscriptionid/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/resourcename",
		Bucket:                    42,
		ClusterResourceGroupIDKey: "/subscriptions/subscriptionid/resourcegroups/clusterresourcegroup",
		ClientIDKey:               "11111111-1111-1111-1111-111111111111",
		OpenShiftCluster: &OpenShiftCluster{
			ID:       "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName",
			Name:     "resourceName",
			Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
			Location: "location",
			Tags: map[string]string{
				"key": "value",
			},
			Properties: OpenShiftClusterProperties{
				ProvisioningState: ProvisioningStateSucceeded,
				ClusterProfile: ClusterProfile{
					PullSecret:      `{"auths":{"registry.connect.redhat.com":{"auth":""},"registry.redhat.io":{"auth":""}}}`,
					Domain:          "cluster.location.aroapp.io",
					Version:         "4.3.0",
					ResourceGroupID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup",
				},
				ConsoleProfile: ConsoleProfile{
					URL: "https://console-openshift-console.apps.cluster.location.aroapp.io/",
				},
				ServicePrincipalProfile: ServicePrincipalProfile{
					TenantID:     "22222222-2222-2222-2222-222222222222",
					ClientSecret: "clientSecret",
					ClientID:     "clientId",
				},
				NetworkProfile: NetworkProfile{
					PodCIDR:     "10.128.0.0/14",
					ServiceCIDR: "172.30.0.0/16",
				},
				MasterProfile: MasterProfile{
					VMSize:   VMSizeStandardD8sV3,
					SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
				},
				WorkerProfiles: []WorkerProfile{
					{
						Name:       "worker",
						VMSize:     VMSizeStandardD2sV3,
						DiskSizeGB: 128,
						SubnetID:   "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
						Count:      3,
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
