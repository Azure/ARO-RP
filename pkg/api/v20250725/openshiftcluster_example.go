package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

func exampleOpenShiftCluster() *OpenShiftCluster {
	doc := api.ExampleOpenShiftClusterDocument()
	return (&openShiftClusterConverter{}).ToExternal(doc.OpenShiftCluster).(*OpenShiftCluster)
}

// ExampleOpenShiftClusterPatchParameter returns an example OpenShiftCluster
// object that an end-user might send to create a cluster in a PATCH request
func ExampleOpenShiftClusterPatchParameter() interface{} {
	oc := ExampleOpenShiftClusterPutParameter().(*OpenShiftCluster)
	oc.Location = nil
	oc.SystemData = nil
	oc.Properties.WorkerProfilesStatus = nil
	oc.Identity = &generated.ManagedServiceIdentity{
		Type: pointerutils.ToPtr(generated.ManagedServiceIdentityTypeUserAssigned),
		UserAssignedIdentities: map[string]*generated.UserAssignedIdentity{
			"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroup/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name": {},
		},
	}
	oc.Properties.PlatformWorkloadIdentityProfile = &generated.PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: map[string]*generated.PlatformWorkloadIdentity{
			"": {
				ResourceID: pointerutils.ToPtr(""),
				ClientID:   pointerutils.ToPtr(""),
				ObjectID:   pointerutils.ToPtr(""),
			},
		},
	}

	return oc
}

// ExampleOpenShiftClusterPutParameter returns an example OpenShiftCluster
// object that an end-user might send to create a cluster in a PUT request
func ExampleOpenShiftClusterPutParameter() interface{} {
	oc := exampleOpenShiftCluster()
	oc.ID = nil
	oc.Name = nil
	oc.Type = nil
	oc.Identity = &generated.ManagedServiceIdentity{
		Type: pointerutils.ToPtr(generated.ManagedServiceIdentityTypeUserAssigned),
		UserAssignedIdentities: map[string]*generated.UserAssignedIdentity{
			"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroup/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name": {},
		},
	}
	oc.Properties.ProvisioningState = nil
	oc.Properties.ClusterProfile.Version = nil
	oc.Properties.ClusterProfile.FipsValidatedModules = pointerutils.ToPtr(generated.FipsValidatedModulesEnabled)
	oc.Properties.ConsoleProfile.URL = nil
	oc.Properties.ApiserverProfile.URL = nil
	oc.Properties.ApiserverProfile.IP = nil
	oc.Properties.IngressProfiles[0].IP = nil
	oc.Properties.MasterProfile.EncryptionAtHost = pointerutils.ToPtr(generated.EncryptionAtHostEnabled)
	oc.Properties.WorkerProfilesStatus = nil
	oc.Properties.NetworkProfile.LoadBalancerProfile = &generated.LoadBalancerProfile{
		ManagedOutboundIPs: &generated.ManagedOutboundIPs{
			Count: pointerutils.ToPtr(int32(1)),
		},
	}
	oc.Properties.PlatformWorkloadIdentityProfile = &generated.PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: map[string]*generated.PlatformWorkloadIdentity{
			"": {
				ResourceID: pointerutils.ToPtr(""),
				ClientID:   pointerutils.ToPtr(""),
				ObjectID:   pointerutils.ToPtr(""),
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
	oc.Properties.ClusterProfile.PullSecret = nil
	oc.Properties.ClusterProfile.OidcIssuer = nil
	oc.Properties.ServicePrincipalProfile.ClientSecret = nil
	oc.Properties.NetworkProfile.LoadBalancerProfile = &generated.LoadBalancerProfile{
		EffectiveOutboundIPs: []*generated.EffectiveOutboundIP{
			{
				ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/clusterResourceGroup/providers/Microsoft.Network/publicIPAddresses/publicIPAddressName"),
			},
		},
		ManagedOutboundIPs: &generated.ManagedOutboundIPs{
			Count: pointerutils.ToPtr(int32(1)),
		},
	}
	oc.Properties.PlatformWorkloadIdentityProfile = &generated.PlatformWorkloadIdentityProfile{
		PlatformWorkloadIdentities: map[string]*generated.PlatformWorkloadIdentity{
			"": {
				ResourceID: pointerutils.ToPtr(""),
				ClientID:   pointerutils.ToPtr(""),
				ObjectID:   pointerutils.ToPtr(""),
			},
		},
	}

	return oc
}

// ExampleOpenShiftClusterResponse returns an example OpenShiftCluster object
// that the RP might return to an end-user in a PUT/PATCH response
func ExampleOpenShiftClusterPutOrPatchResponse() interface{} {
	oc := exampleOpenShiftCluster()
	oc.Properties.ClusterProfile.PullSecret = nil
	oc.Properties.ServicePrincipalProfile.ClientSecret = nil
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
