package v20240812preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleOpenShiftCluster() *OpenShiftCluster {
	doc := api.ExampleOpenShiftClusterDocument()
	return (&openShiftClusterConverter{}).ToExternal(doc.OpenShiftCluster).(*OpenShiftCluster)
}

// ExampleOpenShiftClusterPatchParameter returns an example OpenShiftCluster
// object that an end-user might send to create a cluster in a PATCH request
func ExampleOpenShiftClusterPatchParameter() interface{} {
	oc := ExampleOpenShiftClusterPutParameter().(*OpenShiftCluster)
	oc.Location = ""
	oc.SystemData = nil
	oc.Properties.WorkerProfilesStatus = nil
	oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
			"": {
				ResourceID: "",
				ClientID:   "",
				ObjectID:   "",
			},
		},
	}

	return oc
}

// ExampleOpenShiftClusterPutParameter returns an example OpenShiftCluster
// object that an end-user might send to create a cluster in a PUT request
func ExampleOpenShiftClusterPutParameter() interface{} {
	oc := exampleOpenShiftCluster()
	oc.ID = ""
	oc.Name = ""
	oc.Type = ""
	oc.Identity = &ManagedServiceIdentity{
		Type:                   "",
		UserAssignedIdentities: map[string]UserAssignedIdentity{},
	}
	oc.Properties.ProvisioningState = ""
	oc.Properties.ClusterProfile.Version = ""
	oc.Properties.ClusterProfile.FipsValidatedModules = FipsValidatedModulesEnabled
	oc.Properties.ConsoleProfile.URL = ""
	oc.Properties.APIServerProfile.URL = ""
	oc.Properties.APIServerProfile.IP = ""
	oc.Properties.IngressProfiles[0].IP = ""
	oc.Properties.MasterProfile.EncryptionAtHost = EncryptionAtHostEnabled
	oc.Properties.WorkerProfilesStatus = nil
	oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
		ManagedOutboundIPs: &ManagedOutboundIPs{
			Count: 1,
		},
	}
	oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
			"": {
				ResourceID: "",
				ClientID:   "",
				ObjectID:   "",
			},
		},
	}
	oc.SystemData = nil

	return oc
}

// ExampleOpenShiftClusterResponse returns an example OpenShiftCluster object
// that the RP might return to an end-user in a GET response
func ExampleOpenShiftClusterGetResponse() interface{} {
	oc := exampleOpenShiftCluster()
	oc.Properties.ClusterProfile.PullSecret = ""
	oc.Properties.ClusterProfile.OIDCIssuer = nil
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
	oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
			"": {
				ResourceID: "",
				ClientID:   "",
				ObjectID:   "",
			},
		},
	}

	return oc
}

// ExampleOpenShiftClusterResponse returns an example OpenShiftCluster object
// that the RP might return to an end-user in a PUT/PATCH response
func ExampleOpenShiftClusterPutOrPatchResponse() interface{} {
	oc := exampleOpenShiftCluster()
	oc.Properties.ClusterProfile.PullSecret = ""
	oc.Properties.ServicePrincipalProfile.ClientSecret = ""
	oc.Properties.WorkerProfilesStatus = nil

	return oc
}

// ExampleOpenShiftClusterListResponse returns an example OpenShiftClusterList
// object that the RP might return to an end-user
func ExampleOpenShiftClusterListResponse() interface{} {
	return &OpenShiftClusterList{
		OpenShiftClusters: []*OpenShiftCluster{
			ExampleOpenShiftClusterGetResponse().(*OpenShiftCluster),
		},
	}
}
