package manifest

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func ValidOpenShiftClusterDocument() *api.OpenShiftClusterDocument {
	doc := &api.OpenShiftClusterDocument{}
	doc.OpenShiftCluster = ValidOpenShiftCluster()
	return doc
}

func ValidOpenShiftCluster() *api.OpenShiftCluster {
	return &api.OpenShiftCluster{
		ID:       "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName",
		Name:     "resourceName",
		Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
		Location: "location",
		Tags: map[string]string{
			"key": "value",
		},
		Properties: api.Properties{
			ProvisioningState: api.ProvisioningStateSucceeded,
			ClusterProfile: api.ClusterProfile{
				Domain:          "cluster.location.aroapp.io",
				Version:         "4.3.0",
				ResourceGroupID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup",
			},
			ConsoleProfile: api.ConsoleProfile{
				URL: "https://console-openshift-console.apps.cluster.location.aroapp.io/",
			},
			ServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientSecret: "clientSecret",
				ClientID:     "clientId",
			},
			NetworkProfile: api.NetworkProfile{
				PodCIDR:     "10.128.0.0/14",
				ServiceCIDR: "172.30.0.0/16",
			},
			MasterProfile: api.MasterProfile{
				VMSize:   api.VMSizeStandardD8sV3,
				SubnetID: "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					Name:       "worker",
					VMSize:     api.VMSizeStandardD2sV3,
					DiskSizeGB: 128,
					SubnetID:   "/subscriptions/subscriptionId/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
					Count:      3,
				},
			},
			APIServerProfile: api.APIServerProfile{
				Visibility: api.VisibilityPublic,
				URL:        "https://api.cluster.location.aroapp.io:6443/",
				IP:         "1.2.3.4",
			},
			IngressProfiles: []api.IngressProfile{
				{
					Name:       "default",
					Visibility: api.VisibilityPublic,
					IP:         "1.2.3.4",
				},
			},
		},
	}
}
