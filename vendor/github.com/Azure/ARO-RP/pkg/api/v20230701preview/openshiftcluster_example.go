package v20230701preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleOpenShiftCluster() *OpenShiftCluster {
	doc := api.ExampleOpenShiftClusterDocument()
	doc.OpenShiftCluster.Properties.WorkerProfilesStatus = nil
	return (&openShiftClusterConverter{}).ToExternal(doc.OpenShiftCluster).(*OpenShiftCluster)
}

// ExampleOpenShiftClusterPatchParameter returns an example OpenShiftCluster
// object that an end-user might send to create a cluster in a PATCH request
func ExampleOpenShiftClusterPatchParameter() interface{} {
	oc := ExampleOpenShiftClusterPutParameter().(*OpenShiftCluster)
	oc.Location = ""
	oc.SystemData = nil

	return oc
}

// ExampleOpenShiftClusterPutParameter returns an example OpenShiftCluster
// object that an end-user might send to create a cluster in a PUT request
func ExampleOpenShiftClusterPutParameter() interface{} {
	oc := exampleOpenShiftCluster()
	oc.ID = ""
	oc.Name = ""
	oc.Type = ""
	oc.Properties.ProvisioningState = ""
	oc.Properties.ClusterProfile.Version = ""
	oc.Properties.ClusterProfile.FipsValidatedModules = FipsValidatedModulesEnabled
	oc.Properties.ConsoleProfile.URL = ""
	oc.Properties.APIServerProfile.URL = ""
	oc.Properties.APIServerProfile.IP = ""
	oc.Properties.IngressProfiles[0].IP = ""
	oc.Properties.MasterProfile.EncryptionAtHost = EncryptionAtHostEnabled
	oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
		ManagedOutboundIPs: &ManagedOutboundIPs{
			Count: 1,
		},
	}
	oc.SystemData = nil

	return oc
}

// ExampleOpenShiftClusterResponse returns an example OpenShiftCluster object
// that the RP might return to an end-user
func ExampleOpenShiftClusterResponse() interface{} {
	oc := exampleOpenShiftCluster()
	oc.Properties.ClusterProfile.PullSecret = ""
	oc.Properties.ServicePrincipalProfile.ClientSecret = ""
	oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
		EffectiveOutboundIPs: []EffectiveOutboundIP{
			{
				ID: "/subscriptions/subscriptionId/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/publicIPAddresses/publicIPAddressName",
			},
		},
		ManagedOutboundIPs: &ManagedOutboundIPs{
			Count: 1,
		},
	}

	return oc
}

// ExampleOpenShiftClusterListResponse returns an example OpenShiftClusterList
// object that the RP might return to an end-user
func ExampleOpenShiftClusterListResponse() interface{} {
	return &OpenShiftClusterList{
		OpenShiftClusters: []*OpenShiftCluster{
			ExampleOpenShiftClusterResponse().(*OpenShiftCluster),
		},
	}
}
