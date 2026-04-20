package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

// stripUnusedFields creates a copy of the document with only the fields
// required for monitoring. This significantly reduces memory usage by
// excluding large fields like kubeconfigs, secrets, and other data not
// needed for cluster monitoring.
//
// Fields retained for monitoring:
// - Document metadata: ID, Key, PartitionKey, Bucket
// - Cluster identity: ID, Name, Location, Type
// - Cluster state: ProvisioningState, FailedProvisioningState, ProvisionedBy, CreatedAt
// - Network config: NetworkProfile (for API server IP and NSG checks)
// - Profiles needed: MasterProfile, WorkerProfiles (for subnet monitoring)
// - API access: APIServerProfile, one kubeconfig (AROServiceKubeconfig preferred)
// - Hive integration: HiveProfile
// - Auth type detection: PlatformWorkloadIdentityProfile, ServicePrincipalProfile (presence only)
func stripUnusedFields(doc *api.OpenShiftClusterDocument) *api.OpenShiftClusterDocument {
	if doc == nil || doc.OpenShiftCluster == nil {
		return doc
	}

	oc := doc.OpenShiftCluster

	// Select the kubeconfig to keep - prefer AROServiceKubeconfig
	var kubeconfigToKeep api.SecureBytes
	if oc.Properties.AROServiceKubeconfig != nil {
		kubeconfigToKeep = oc.Properties.AROServiceKubeconfig
	} else {
		kubeconfigToKeep = oc.Properties.AdminKubeconfig
	}

	// Create stripped worker profiles (only need Name, SubnetID, Count)
	var strippedWorkerProfiles []api.WorkerProfile
	if oc.Properties.WorkerProfiles != nil {
		strippedWorkerProfiles = make([]api.WorkerProfile, len(oc.Properties.WorkerProfiles))
		for i, wp := range oc.Properties.WorkerProfiles {
			strippedWorkerProfiles[i] = api.WorkerProfile{
				Name:     wp.Name,
				SubnetID: wp.SubnetID,
				Count:    wp.Count,
			}
		}
	}

	// Create minimal ServicePrincipalProfile (only need to check presence, not credentials)
	var strippedSPProfile *api.ServicePrincipalProfile
	if oc.Properties.ServicePrincipalProfile != nil {
		strippedSPProfile = &api.ServicePrincipalProfile{
			ClientID: oc.Properties.ServicePrincipalProfile.ClientID,
			// Intentionally omit ClientSecret - not needed for monitoring
		}
	}

	// Create minimal PlatformWorkloadIdentityProfile (only need to check presence)
	var strippedPWIProfile *api.PlatformWorkloadIdentityProfile
	if oc.Properties.PlatformWorkloadIdentityProfile != nil {
		strippedPWIProfile = &api.PlatformWorkloadIdentityProfile{}
	}

	// Build the stripped document
	stripped := &api.OpenShiftClusterDocument{
		ID:           doc.ID,
		Key:          doc.Key,
		PartitionKey: doc.PartitionKey,
		Bucket:       doc.Bucket,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:       oc.ID,
			Name:     oc.Name,
			Type:     oc.Type,
			Location: oc.Location,
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState:       oc.Properties.ProvisioningState,
				FailedProvisioningState: oc.Properties.FailedProvisioningState,
				ProvisionedBy:           oc.Properties.ProvisionedBy,
				CreatedAt:               oc.Properties.CreatedAt,
				ClusterProfile: api.ClusterProfile{
					Domain:  oc.Properties.ClusterProfile.Domain,
					Version: oc.Properties.ClusterProfile.Version,
					// Intentionally omit PullSecret, BoundServiceAccountSigningKey
				},
				NetworkProfile: api.NetworkProfile{
					PodCIDR:                    oc.Properties.NetworkProfile.PodCIDR,
					ServiceCIDR:                oc.Properties.NetworkProfile.ServiceCIDR,
					APIServerPrivateEndpointIP: oc.Properties.NetworkProfile.APIServerPrivateEndpointIP,
					PreconfiguredNSG:           oc.Properties.NetworkProfile.PreconfiguredNSG,
				},
				MasterProfile: api.MasterProfile{
					SubnetID: oc.Properties.MasterProfile.SubnetID,
				},
				WorkerProfiles: strippedWorkerProfiles,
				APIServerProfile: api.APIServerProfile{
					Visibility: oc.Properties.APIServerProfile.Visibility,
					URL:        oc.Properties.APIServerProfile.URL,
					IP:         oc.Properties.APIServerProfile.IP,
				},
				AROServiceKubeconfig:            kubeconfigToKeep,
				HiveProfile:                     oc.Properties.HiveProfile,
				ServicePrincipalProfile:         strippedSPProfile,
				PlatformWorkloadIdentityProfile: strippedPWIProfile,
				MaintenanceState:                oc.Properties.MaintenanceState,
			},
		},
	}

	return stripped
}
