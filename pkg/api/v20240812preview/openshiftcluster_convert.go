package v20240812preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

type openShiftClusterConverter struct{}

// ToExternal returns a new external representation of the internal object,
// reading from the subset of the internal object's fields that appear in the
// external representation.  ToExternal does not modify its argument; there is
// no pointer aliasing between the passed and returned objects
func (c openShiftClusterConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {
	out := &OpenShiftCluster{
		ID:       oc.ID,
		Name:     oc.Name,
		Type:     oc.Type,
		Location: oc.Location,
		Properties: OpenShiftClusterProperties{
			ProvisioningState: ProvisioningState(oc.Properties.ProvisioningState),
			ClusterProfile: ClusterProfile{
				PullSecret:           string(oc.Properties.ClusterProfile.PullSecret),
				Domain:               oc.Properties.ClusterProfile.Domain,
				Version:              oc.Properties.ClusterProfile.Version,
				ResourceGroupID:      oc.Properties.ClusterProfile.ResourceGroupID,
				FipsValidatedModules: FipsValidatedModules(oc.Properties.ClusterProfile.FipsValidatedModules),
			},
			ConsoleProfile: ConsoleProfile{
				URL: oc.Properties.ConsoleProfile.URL,
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:          oc.Properties.NetworkProfile.PodCIDR,
				ServiceCIDR:      oc.Properties.NetworkProfile.ServiceCIDR,
				OutboundType:     OutboundType(oc.Properties.NetworkProfile.OutboundType),
				PreconfiguredNSG: PreconfiguredNSG(oc.Properties.NetworkProfile.PreconfiguredNSG),
			},
			MasterProfile: MasterProfile{
				VMSize:              VMSize(oc.Properties.MasterProfile.VMSize),
				SubnetID:            oc.Properties.MasterProfile.SubnetID,
				EncryptionAtHost:    EncryptionAtHost(oc.Properties.MasterProfile.EncryptionAtHost),
				DiskEncryptionSetID: oc.Properties.MasterProfile.DiskEncryptionSetID,
			},
			APIServerProfile: APIServerProfile{
				Visibility: Visibility(oc.Properties.APIServerProfile.Visibility),
				URL:        oc.Properties.APIServerProfile.URL,
				IP:         oc.Properties.APIServerProfile.IP,
			},
		},
	}

	if oc.Properties.ServicePrincipalProfile != nil {
		out.Properties.ServicePrincipalProfile = &ServicePrincipalProfile{
			ClientID:     oc.Properties.ServicePrincipalProfile.ClientID,
			ClientSecret: string(oc.Properties.ServicePrincipalProfile.ClientSecret),
		}
	}

	if oc.Properties.NetworkProfile.LoadBalancerProfile != nil {
		out.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{}

		if oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
			out.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs = &ManagedOutboundIPs{
				Count: oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count,
			}
		}

		if oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs != nil {
			out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = make([]EffectiveOutboundIP, 0, len(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs))
			for _, effectiveOutboundIP := range oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs {
				out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = append(out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs, EffectiveOutboundIP{
					ID: effectiveOutboundIP.ID,
				})
			}
		}
	}

	if oc.Properties.WorkerProfiles != nil {
		workerProfiles := oc.Properties.WorkerProfiles
		out.Properties.WorkerProfiles = make([]WorkerProfile, 0, len(workerProfiles))
		for _, p := range workerProfiles {
			out.Properties.WorkerProfiles = append(out.Properties.WorkerProfiles, WorkerProfile{
				Name:                p.Name,
				VMSize:              VMSize(p.VMSize),
				DiskSizeGB:          p.DiskSizeGB,
				SubnetID:            p.SubnetID,
				Count:               p.Count,
				EncryptionAtHost:    EncryptionAtHost(p.EncryptionAtHost),
				DiskEncryptionSetID: p.DiskEncryptionSetID,
			})
		}
	}

	if oc.Properties.WorkerProfilesStatus != nil {
		workerProfiles := oc.Properties.WorkerProfilesStatus
		out.Properties.WorkerProfilesStatus = make([]WorkerProfile, 0, len(workerProfiles))
		for _, p := range workerProfiles {
			out.Properties.WorkerProfilesStatus = append(out.Properties.WorkerProfilesStatus, WorkerProfile{
				Name:                p.Name,
				VMSize:              VMSize(p.VMSize),
				DiskSizeGB:          p.DiskSizeGB,
				SubnetID:            p.SubnetID,
				Count:               p.Count,
				EncryptionAtHost:    EncryptionAtHost(p.EncryptionAtHost),
				DiskEncryptionSetID: p.DiskEncryptionSetID,
			})
		}
	}

	if oc.Properties.IngressProfiles != nil {
		out.Properties.IngressProfiles = make([]IngressProfile, 0, len(oc.Properties.IngressProfiles))
		for _, p := range oc.Properties.IngressProfiles {
			out.Properties.IngressProfiles = append(out.Properties.IngressProfiles, IngressProfile{
				Name:       p.Name,
				Visibility: Visibility(p.Visibility),
				IP:         p.IP,
			})
		}
	}

	if oc.Tags != nil {
		out.Tags = make(map[string]string, len(oc.Tags))
		for k, v := range oc.Tags {
			out.Tags[k] = v
		}
	}

	if oc.Identity != nil {
		out.Identity = &ManagedServiceIdentity{}
		out.Identity.Type = ManagedServiceIdentityType(oc.Identity.Type)
		out.Identity.PrincipalID = oc.Identity.PrincipalID
		out.Identity.TenantID = oc.Identity.TenantID
		out.Identity.UserAssignedIdentities = make(map[string]UserAssignedIdentity, len(oc.Identity.UserAssignedIdentities))
		for k := range oc.Identity.UserAssignedIdentities {
			var temp UserAssignedIdentity
			temp.ClientID = oc.Identity.UserAssignedIdentities[k].ClientID
			temp.PrincipalID = oc.Identity.UserAssignedIdentities[k].PrincipalID
			out.Identity.UserAssignedIdentities[k] = temp
		}
	}

	if oc.Properties.PlatformWorkloadIdentityProfile != nil && oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities != nil {
		out.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{}

		if oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo != nil {
			temp := UpgradeableTo(*oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo)
			out.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = &temp
		}

		out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = make(map[string]PlatformWorkloadIdentity, len(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities))

		for k := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			pwi := PlatformWorkloadIdentity{
				ClientID:   oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k].ClientID,
				ObjectID:   oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k].ObjectID,
				ResourceID: oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k].ResourceID,
			}

			out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k] = pwi
		}
	}

	if oc.Properties.ClusterProfile.OIDCIssuer != nil {
		out.Properties.ClusterProfile.OIDCIssuer = pointerutils.ToPtr(OIDCIssuer(*oc.Properties.ClusterProfile.OIDCIssuer))
	}

	out.SystemData = &SystemData{
		CreatedBy:          oc.SystemData.CreatedBy,
		CreatedAt:          oc.SystemData.CreatedAt,
		CreatedByType:      CreatedByType(oc.SystemData.CreatedByType),
		LastModifiedBy:     oc.SystemData.LastModifiedBy,
		LastModifiedAt:     oc.SystemData.LastModifiedAt,
		LastModifiedByType: CreatedByType(oc.SystemData.LastModifiedByType),
	}

	return out
}

// ToExternalList returns a slice of external representations of the internal
// objects
func (c openShiftClusterConverter) ToExternalList(ocs []*api.OpenShiftCluster, nextLink string) interface{} {
	l := &OpenShiftClusterList{
		OpenShiftClusters: make([]*OpenShiftCluster, 0, len(ocs)),
		NextLink:          nextLink,
	}

	for _, oc := range ocs {
		l.OpenShiftClusters = append(l.OpenShiftClusters, c.ToExternal(oc).(*OpenShiftCluster))
	}

	return l
}

// ToInternal overwrites in place a pre-existing internal object, setting (only)
// all mapped fields from the external representation. ToInternal modifies its
// argument; there is no pointer aliasing between the passed and returned
// objects
func (c openShiftClusterConverter) ToInternal(_oc interface{}, out *api.OpenShiftCluster) {
	oc := _oc.(*OpenShiftCluster)

	out.ID = oc.ID
	out.Name = oc.Name
	out.Type = oc.Type
	out.Location = oc.Location
	out.Tags = nil
	if oc.Tags != nil {
		out.Tags = make(map[string]string, len(oc.Tags))
		for k, v := range oc.Tags {
			out.Tags[k] = v
		}
	}

	if oc.Identity != nil {
		if out.Identity == nil {
			out.Identity = &api.ManagedServiceIdentity{}
		}
		out.Identity.Type = api.ManagedServiceIdentityType(oc.Identity.Type)
		out.Identity.PrincipalID = oc.Identity.PrincipalID
		out.Identity.TenantID = oc.Identity.TenantID
		out.Identity.UserAssignedIdentities = make(map[string]api.UserAssignedIdentity, len(oc.Identity.UserAssignedIdentities))
		for k := range oc.Identity.UserAssignedIdentities {
			var temp api.UserAssignedIdentity
			temp.ClientID = oc.Identity.UserAssignedIdentities[k].ClientID
			temp.PrincipalID = oc.Identity.UserAssignedIdentities[k].PrincipalID
			out.Identity.UserAssignedIdentities[k] = temp
		}
	}

	out.Properties.ProvisioningState = api.ProvisioningState(oc.Properties.ProvisioningState)
	out.Properties.ClusterProfile.PullSecret = api.SecureString(oc.Properties.ClusterProfile.PullSecret)
	out.Properties.ClusterProfile.Domain = oc.Properties.ClusterProfile.Domain
	out.Properties.ClusterProfile.Version = oc.Properties.ClusterProfile.Version
	out.Properties.ClusterProfile.ResourceGroupID = oc.Properties.ClusterProfile.ResourceGroupID
	if oc.Properties.ConsoleProfile.URL != "" {
		out.Properties.ConsoleProfile.URL = oc.Properties.ConsoleProfile.URL
	}
	out.Properties.ClusterProfile.FipsValidatedModules = api.FipsValidatedModules(oc.Properties.ClusterProfile.FipsValidatedModules)
	if oc.Properties.ServicePrincipalProfile != nil {
		out.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{
			ClientID:     oc.Properties.ServicePrincipalProfile.ClientID,
			ClientSecret: api.SecureString(oc.Properties.ServicePrincipalProfile.ClientSecret),
		}
	}
	if oc.Properties.PlatformWorkloadIdentityProfile != nil && oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities != nil {
		if out.Properties.PlatformWorkloadIdentityProfile == nil {
			out.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
		}

		if oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo != nil {
			temp := api.UpgradeableTo(*oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo)
			out.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = &temp
		}

		if out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities == nil {
			out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = make(map[string]api.PlatformWorkloadIdentity, len(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities))
		}

		for k, identity := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			if pwi, exists := out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k]; exists {
				if pwi.ResourceID != identity.ResourceID {
					pwi.ClientID = ""
					pwi.ObjectID = ""
				}
				pwi.ResourceID = identity.ResourceID
				out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k] = pwi
			} else {
				out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k] = api.PlatformWorkloadIdentity{
					ResourceID: identity.ResourceID,
				}
			}
		}
	}

	out.Properties.NetworkProfile.PodCIDR = oc.Properties.NetworkProfile.PodCIDR
	out.Properties.NetworkProfile.ServiceCIDR = oc.Properties.NetworkProfile.ServiceCIDR
	out.Properties.NetworkProfile.OutboundType = api.OutboundType(oc.Properties.NetworkProfile.OutboundType)
	out.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSG(oc.Properties.NetworkProfile.PreconfiguredNSG)

	if oc.Properties.NetworkProfile.LoadBalancerProfile != nil {
		loadBalancerProfile := api.LoadBalancerProfile{}

		// EffectiveOutboundIPs is a read-only field, so it will never be present in requests.
		// Preserve the slice from the pre-existing internal object.
		if out.Properties.NetworkProfile.LoadBalancerProfile != nil {
			loadBalancerProfile.EffectiveOutboundIPs = make([]api.EffectiveOutboundIP, len(out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs))
			copy(loadBalancerProfile.EffectiveOutboundIPs, out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs)
		}

		out.Properties.NetworkProfile.LoadBalancerProfile = &loadBalancerProfile

		if oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
			out.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs = &api.ManagedOutboundIPs{
				Count: oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count,
			}
		}
		if oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs != nil {
			out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = make([]api.EffectiveOutboundIP, len(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs))
			for i := range oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs {
				out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs[i].ID = oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs[i].ID
			}
		}
	}

	out.Properties.MasterProfile.VMSize = api.VMSize(oc.Properties.MasterProfile.VMSize)
	out.Properties.MasterProfile.SubnetID = oc.Properties.MasterProfile.SubnetID
	out.Properties.MasterProfile.EncryptionAtHost = api.EncryptionAtHost(oc.Properties.MasterProfile.EncryptionAtHost)
	out.Properties.MasterProfile.DiskEncryptionSetID = oc.Properties.MasterProfile.DiskEncryptionSetID
	out.Properties.WorkerProfiles = nil
	if oc.Properties.WorkerProfiles != nil {
		out.Properties.WorkerProfiles = make([]api.WorkerProfile, len(oc.Properties.WorkerProfiles))
		for i := range oc.Properties.WorkerProfiles {
			out.Properties.WorkerProfiles[i].Name = oc.Properties.WorkerProfiles[i].Name
			out.Properties.WorkerProfiles[i].VMSize = api.VMSize(oc.Properties.WorkerProfiles[i].VMSize)
			out.Properties.WorkerProfiles[i].DiskSizeGB = oc.Properties.WorkerProfiles[i].DiskSizeGB
			out.Properties.WorkerProfiles[i].SubnetID = oc.Properties.WorkerProfiles[i].SubnetID
			out.Properties.WorkerProfiles[i].Count = oc.Properties.WorkerProfiles[i].Count
			out.Properties.WorkerProfiles[i].EncryptionAtHost = api.EncryptionAtHost(oc.Properties.WorkerProfiles[i].EncryptionAtHost)
			out.Properties.WorkerProfiles[i].DiskEncryptionSetID = oc.Properties.WorkerProfiles[i].DiskEncryptionSetID
		}
	}
	out.Properties.WorkerProfilesStatus = nil
	if oc.Properties.WorkerProfilesStatus != nil {
		out.Properties.WorkerProfilesStatus = make([]api.WorkerProfile, len(oc.Properties.WorkerProfilesStatus))
		for i := range oc.Properties.WorkerProfilesStatus {
			out.Properties.WorkerProfilesStatus[i].Name = oc.Properties.WorkerProfilesStatus[i].Name
			out.Properties.WorkerProfilesStatus[i].VMSize = api.VMSize(oc.Properties.WorkerProfilesStatus[i].VMSize)
			out.Properties.WorkerProfilesStatus[i].DiskSizeGB = oc.Properties.WorkerProfilesStatus[i].DiskSizeGB
			out.Properties.WorkerProfilesStatus[i].SubnetID = oc.Properties.WorkerProfilesStatus[i].SubnetID
			out.Properties.WorkerProfilesStatus[i].Count = oc.Properties.WorkerProfilesStatus[i].Count
			out.Properties.WorkerProfilesStatus[i].EncryptionAtHost = api.EncryptionAtHost(oc.Properties.WorkerProfilesStatus[i].EncryptionAtHost)
			out.Properties.WorkerProfilesStatus[i].DiskEncryptionSetID = oc.Properties.WorkerProfilesStatus[i].DiskEncryptionSetID
		}
	}
	out.Properties.APIServerProfile.Visibility = api.Visibility(oc.Properties.APIServerProfile.Visibility)
	if oc.Properties.APIServerProfile.URL != "" {
		out.Properties.APIServerProfile.URL = oc.Properties.APIServerProfile.URL
	}
	if oc.Properties.APIServerProfile.IP != "" {
		out.Properties.APIServerProfile.IP = oc.Properties.APIServerProfile.IP
	}
	out.Properties.IngressProfiles = nil
	if oc.Properties.IngressProfiles != nil {
		out.Properties.IngressProfiles = make([]api.IngressProfile, len(oc.Properties.IngressProfiles))
		for i := range oc.Properties.IngressProfiles {
			out.Properties.IngressProfiles[i].Name = oc.Properties.IngressProfiles[i].Name
			out.Properties.IngressProfiles[i].Visibility = api.Visibility(oc.Properties.IngressProfiles[i].Visibility)
			if oc.Properties.IngressProfiles[i].IP != "" {
				out.Properties.IngressProfiles[i].IP = oc.Properties.IngressProfiles[i].IP
			}
		}
	}

	if oc.SystemData != nil {
		out.SystemData = api.SystemData{
			CreatedBy:          oc.SystemData.CreatedBy,
			CreatedAt:          oc.SystemData.CreatedAt,
			CreatedByType:      api.CreatedByType(oc.SystemData.CreatedByType),
			LastModifiedBy:     oc.SystemData.LastModifiedBy,
			LastModifiedAt:     oc.SystemData.LastModifiedAt,
			LastModifiedByType: api.CreatedByType(oc.SystemData.CreatedByType),
		}
	}
}

// ExternalNoReadOnly removes all read-only fields from the external representation.
func (c openShiftClusterConverter) ExternalNoReadOnly(_oc interface{}) {
	oc := _oc.(*OpenShiftCluster)
	oc.Properties.WorkerProfilesStatus = nil
	if oc.Properties.NetworkProfile.LoadBalancerProfile != nil {
		oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = nil
	}
	oc.SystemData = nil
	oc.Properties.ConsoleProfile.URL = ""
	oc.Properties.APIServerProfile.URL = ""
	oc.Properties.APIServerProfile.IP = ""
	for i := range oc.Properties.IngressProfiles {
		oc.Properties.IngressProfiles[i].IP = ""
	}
	if oc.Properties.PlatformWorkloadIdentityProfile != nil {
		for i := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			if entry, ok := oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[i]; ok {
				entry.ClientID = ""
				entry.ObjectID = ""
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[i] = entry
			}
		}
	}
	if oc.Identity != nil {
		oc.Identity.PrincipalID = ""
		oc.Identity.TenantID = ""
		for i := range oc.Identity.UserAssignedIdentities {
			if entry, ok := oc.Identity.UserAssignedIdentities[i]; ok {
				entry.ClientID = ""
				entry.PrincipalID = ""
				oc.Identity.UserAssignedIdentities[i] = entry
			}
		}
	}
}
