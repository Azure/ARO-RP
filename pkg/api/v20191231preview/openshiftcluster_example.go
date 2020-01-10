package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func exampleOpenShiftCluster() *OpenShiftCluster {
	return &OpenShiftCluster{
		ID:       "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName",
		Name:     "resourceName",
		Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
		Location: "location",
		Tags: Tags{
			"key": "value",
		},
		Properties: Properties{
			ProvisioningState: ProvisioningStateSucceeded,
			ClusterProfile: ClusterProfile{
				Domain: "cluster.location.aroapp.io",
			},
			ServicePrincipalProfile: ServicePrincipalProfile{
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
			ConsoleURL: "https://console-openshift-console.apps.cluster.location.aroapp.io/",
		},
	}
}

// ExampleOpenShiftClusterParameter returns an example OpenShiftCluster object
// that an end-user might send to create a cluster in a PUT or PATCH request
func ExampleOpenShiftClusterParameter() *OpenShiftCluster {
	oc := exampleOpenShiftCluster()
	oc.ID = ""
	oc.Name = ""
	oc.Type = ""
	oc.Properties.ProvisioningState = ""
	oc.Properties.APIServerProfile.URL = ""
	oc.Properties.APIServerProfile.IP = ""
	oc.Properties.IngressProfiles[0].IP = ""
	oc.Properties.ConsoleURL = ""

	return oc
}

// ExampleOpenShiftClusterResponse returns an example OpenShiftCluster object
// that the RP might return to an end-user
func ExampleOpenShiftClusterResponse() *OpenShiftCluster {
	oc := exampleOpenShiftCluster()
	oc.Properties.ServicePrincipalProfile.ClientSecret = ""

	return oc
}

// ExampleOpenShiftClusterListResponse returns an example OpenShiftClusterList
// object that the RP might return to an end-user
func ExampleOpenShiftClusterListResponse() *OpenShiftClusterList {
	return &OpenShiftClusterList{
		OpenShiftClusters: []*OpenShiftCluster{
			ExampleOpenShiftClusterResponse(),
		},
	}
}
