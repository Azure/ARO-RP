package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

type openShiftClusterConverter struct{}

// toPtrIfNonZero returns a pointer to the value if it is non-zero and returns nil otherwise.
// Since the generated API models use pointer types for all fields, this helper function ensures
// that pointers to zero values are excluded from API responses. In other words, it helps ensure
// that *effectively* empty fields are excluded from API responses.
func toPtrIfNonZero[T comparable](value T) *T {
	var zero T
	if value == zero {
		return nil
	}
	return pointerutils.ToPtr(value)
}

// ToExternal returns a new external representation of the internal object,
// reading from the subset of the internal object's fields that appear in the
// external representation.  ToExternal does not modify its argument; there is
// no pointer aliasing between the passed and returned objects
func (c openShiftClusterConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {
	out := &OpenShiftCluster{
		OpenShiftCluster: generated.OpenShiftCluster{
			ID:       toPtrIfNonZero(oc.ID),
			Name:     toPtrIfNonZero(oc.Name),
			Type:     toPtrIfNonZero(oc.Type),
			Location: toPtrIfNonZero(oc.Location),
			Properties: &generated.OpenShiftClusterProperties{
				ProvisioningState: toPtrIfNonZero(generated.ProvisioningState(oc.Properties.ProvisioningState)),
				ClusterProfile: &generated.ClusterProfile{
					Domain:               toPtrIfNonZero(oc.Properties.ClusterProfile.Domain),
					Version:              toPtrIfNonZero(oc.Properties.ClusterProfile.Version),
					ResourceGroupID:      toPtrIfNonZero(oc.Properties.ClusterProfile.ResourceGroupID),
					FipsValidatedModules: toPtrIfNonZero(generated.FipsValidatedModules(oc.Properties.ClusterProfile.FipsValidatedModules)),
				},
				NetworkProfile: &generated.NetworkProfile{
					PodCidr:          toPtrIfNonZero(oc.Properties.NetworkProfile.PodCIDR),
					ServiceCidr:      toPtrIfNonZero(oc.Properties.NetworkProfile.ServiceCIDR),
					OutboundType:     toPtrIfNonZero(generated.OutboundType(oc.Properties.NetworkProfile.OutboundType)),
					PreconfiguredNSG: toPtrIfNonZero(generated.PreconfiguredNSG(oc.Properties.NetworkProfile.PreconfiguredNSG)),
				},
				MasterProfile: &generated.MasterProfile{
					VMSize:              toPtrIfNonZero(string(oc.Properties.MasterProfile.VMSize)),
					SubnetID:            toPtrIfNonZero(oc.Properties.MasterProfile.SubnetID),
					EncryptionAtHost:    toPtrIfNonZero(generated.EncryptionAtHost(oc.Properties.MasterProfile.EncryptionAtHost)),
					DiskEncryptionSetID: toPtrIfNonZero(oc.Properties.MasterProfile.DiskEncryptionSetID),
				},
				ApiserverProfile: &generated.APIServerProfile{
					Visibility: toPtrIfNonZero(generated.Visibility(oc.Properties.APIServerProfile.Visibility)),
					URL:        toPtrIfNonZero(oc.Properties.APIServerProfile.URL),
					IP:         toPtrIfNonZero(oc.Properties.APIServerProfile.IP),
				},
			},
		},
	}

	if oc.Properties.ConsoleProfile.URL != "" {
		out.Properties.ConsoleProfile = &generated.ConsoleProfile{
			URL: pointerutils.ToPtr(oc.Properties.ConsoleProfile.URL),
		}
	}

	if oc.Properties.ClusterProfile.PullSecret != "" {
		out.Properties.ClusterProfile.PullSecret = pointerutils.ToPtr(string(oc.Properties.ClusterProfile.PullSecret))
	}

	if oc.Properties.ServicePrincipalProfile != nil {
		out.Properties.ServicePrincipalProfile = &generated.ServicePrincipalProfile{
			ClientID: toPtrIfNonZero(oc.Properties.ServicePrincipalProfile.ClientID),
		}
		if oc.Properties.ServicePrincipalProfile.ClientSecret != "" {
			out.Properties.ServicePrincipalProfile.ClientSecret = pointerutils.ToPtr(string(oc.Properties.ServicePrincipalProfile.ClientSecret))
		}
	}

	if oc.Properties.NetworkProfile.LoadBalancerProfile != nil {
		out.Properties.NetworkProfile.LoadBalancerProfile = &generated.LoadBalancerProfile{}

		if oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
			out.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs = &generated.ManagedOutboundIPs{
				Count: toPtrIfNonZero(int32(oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count)),
			}
		}

		if oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs != nil {
			out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = make([]*generated.EffectiveOutboundIP, 0, len(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs))
			for _, effectiveOutboundIP := range oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs {
				out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = append(out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs, &generated.EffectiveOutboundIP{
					ID: toPtrIfNonZero(effectiveOutboundIP.ID),
				})
			}
		}
	}

	if oc.Properties.WorkerProfiles != nil {
		workerProfiles := oc.Properties.WorkerProfiles
		out.Properties.WorkerProfiles = make([]*generated.WorkerProfile, 0, len(workerProfiles))
		for _, p := range workerProfiles {
			out.Properties.WorkerProfiles = append(out.Properties.WorkerProfiles, &generated.WorkerProfile{
				Name:                toPtrIfNonZero(p.Name),
				VMSize:              toPtrIfNonZero(string(p.VMSize)),
				DiskSizeGB:          toPtrIfNonZero(int32(p.DiskSizeGB)),
				SubnetID:            toPtrIfNonZero(p.SubnetID),
				Count:               toPtrIfNonZero(int32(p.Count)),
				EncryptionAtHost:    toPtrIfNonZero(generated.EncryptionAtHost(p.EncryptionAtHost)),
				DiskEncryptionSetID: toPtrIfNonZero(p.DiskEncryptionSetID),
			})
		}
	}

	if oc.Properties.WorkerProfilesStatus != nil {
		workerProfiles := oc.Properties.WorkerProfilesStatus
		out.Properties.WorkerProfilesStatus = make([]*generated.WorkerProfile, 0, len(workerProfiles))
		for _, p := range workerProfiles {
			out.Properties.WorkerProfilesStatus = append(out.Properties.WorkerProfilesStatus, &generated.WorkerProfile{
				Name:                toPtrIfNonZero(p.Name),
				VMSize:              toPtrIfNonZero(string(p.VMSize)),
				DiskSizeGB:          toPtrIfNonZero(int32(p.DiskSizeGB)),
				SubnetID:            toPtrIfNonZero(p.SubnetID),
				Count:               toPtrIfNonZero(int32(p.Count)),
				EncryptionAtHost:    toPtrIfNonZero(generated.EncryptionAtHost(p.EncryptionAtHost)),
				DiskEncryptionSetID: toPtrIfNonZero(p.DiskEncryptionSetID),
			})
		}
	}

	if oc.Properties.IngressProfiles != nil {
		out.Properties.IngressProfiles = make([]*generated.IngressProfile, 0, len(oc.Properties.IngressProfiles))
		for _, p := range oc.Properties.IngressProfiles {
			out.Properties.IngressProfiles = append(out.Properties.IngressProfiles, &generated.IngressProfile{
				Name:       toPtrIfNonZero(p.Name),
				Visibility: toPtrIfNonZero(generated.Visibility(p.Visibility)),
				IP:         toPtrIfNonZero(p.IP),
			})
		}
	}

	if len(oc.Tags) > 0 {
		out.Tags = make(map[string]*string, len(oc.Tags))
		for k, v := range oc.Tags {
			out.Tags[k] = pointerutils.ToPtr(v)
		}
	}

	if oc.Identity != nil {
		out.Identity = &generated.ManagedServiceIdentity{}
		out.Identity.Type = toPtrIfNonZero(generated.ManagedServiceIdentityType(oc.Identity.Type))
		out.Identity.PrincipalID = toPtrIfNonZero(oc.Identity.PrincipalID)
		out.Identity.TenantID = toPtrIfNonZero(oc.Identity.TenantID)
		if len(oc.Identity.UserAssignedIdentities) > 0 {
			out.Identity.UserAssignedIdentities = make(map[string]*generated.UserAssignedIdentity, len(oc.Identity.UserAssignedIdentities))
			for k := range oc.Identity.UserAssignedIdentities {
				temp := &generated.UserAssignedIdentity{}
				temp.ClientID = toPtrIfNonZero(oc.Identity.UserAssignedIdentities[k].ClientID)
				temp.PrincipalID = toPtrIfNonZero(oc.Identity.UserAssignedIdentities[k].PrincipalID)
				out.Identity.UserAssignedIdentities[k] = temp
			}
		}
	}

	if oc.Properties.PlatformWorkloadIdentityProfile != nil && oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities != nil {
		out.Properties.PlatformWorkloadIdentityProfile = &generated.PlatformWorkloadIdentityProfile{}

		if oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo != nil {
			temp := string(*oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo)
			out.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo = &temp
		}

		if len(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities) > 0 {
			out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = make(map[string]*generated.PlatformWorkloadIdentity, len(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities))

			for k := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
				pwi := &generated.PlatformWorkloadIdentity{
					ClientID:   toPtrIfNonZero(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k].ClientID),
					ObjectID:   toPtrIfNonZero(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k].ObjectID),
					ResourceID: toPtrIfNonZero(oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k].ResourceID),
				}

				out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k] = pwi
			}
		}
	}

	if oc.Properties.ClusterProfile.OIDCIssuer != nil {
		out.Properties.ClusterProfile.OidcIssuer = pointerutils.ToPtr(string(*oc.Properties.ClusterProfile.OIDCIssuer))
	}

	out.SystemData = &generated.SystemData{
		CreatedBy:          toPtrIfNonZero(oc.SystemData.CreatedBy),
		CreatedAt:          oc.SystemData.CreatedAt,
		CreatedByType:      toPtrIfNonZero(generated.CreatedByType(oc.SystemData.CreatedByType)),
		LastModifiedBy:     toPtrIfNonZero(oc.SystemData.LastModifiedBy),
		LastModifiedAt:     oc.SystemData.LastModifiedAt,
		LastModifiedByType: toPtrIfNonZero(generated.CreatedByType(oc.SystemData.LastModifiedByType)),
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

	out.ID = value(oc.ID)
	out.Name = value(oc.Name)
	out.Type = value(oc.Type)
	out.Location = value(oc.Location)
	out.Tags = nil
	if oc.Tags != nil {
		out.Tags = make(map[string]string, len(oc.Tags))
		for k, v := range oc.Tags {
			out.Tags[k] = value(v)
		}
	}

	if oc.Identity != nil {
		if out.Identity == nil {
			out.Identity = &api.ManagedServiceIdentity{}
		}
		out.Identity.Type = api.ManagedServiceIdentityType(value(oc.Identity.Type))
		out.Identity.PrincipalID = value(oc.Identity.PrincipalID)
		out.Identity.TenantID = value(oc.Identity.TenantID)
		out.Identity.UserAssignedIdentities = make(map[string]api.UserAssignedIdentity, len(oc.Identity.UserAssignedIdentities))
		for k := range oc.Identity.UserAssignedIdentities {
			var temp api.UserAssignedIdentity
			if oc.Identity.UserAssignedIdentities[k] != nil {
				temp.ClientID = value(oc.Identity.UserAssignedIdentities[k].ClientID)
				temp.PrincipalID = value(oc.Identity.UserAssignedIdentities[k].PrincipalID)
			}
			out.Identity.UserAssignedIdentities[k] = temp
		}
	}

	out.Properties.ProvisioningState = api.ProvisioningState(value(oc.Properties.ProvisioningState))
	out.Properties.ClusterProfile.PullSecret = api.SecureString(value(oc.Properties.ClusterProfile.PullSecret))
	out.Properties.ClusterProfile.Domain = value(oc.Properties.ClusterProfile.Domain)
	out.Properties.ClusterProfile.Version = value(oc.Properties.ClusterProfile.Version)
	out.Properties.ClusterProfile.ResourceGroupID = value(oc.Properties.ClusterProfile.ResourceGroupID)
	if oc.Properties.ConsoleProfile != nil && value(oc.Properties.ConsoleProfile.URL) != "" {
		out.Properties.ConsoleProfile.URL = value(oc.Properties.ConsoleProfile.URL)
	}
	out.Properties.ClusterProfile.FipsValidatedModules = api.FipsValidatedModules(value(oc.Properties.ClusterProfile.FipsValidatedModules))
	if oc.Properties.ServicePrincipalProfile != nil {
		out.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{
			ClientID:     value(oc.Properties.ServicePrincipalProfile.ClientID),
			ClientSecret: api.SecureString(value(oc.Properties.ServicePrincipalProfile.ClientSecret)),
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
			if identity == nil {
				continue
			}
			if pwi, exists := out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k]; exists {
				if pwi.ResourceID != value(identity.ResourceID) {
					pwi.ClientID = ""
					pwi.ObjectID = ""
				}
				pwi.ResourceID = value(identity.ResourceID)
				out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k] = pwi
			} else {
				out.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[k] = api.PlatformWorkloadIdentity{
					ResourceID: value(identity.ResourceID),
				}
			}
		}
	}

	out.Properties.NetworkProfile.PodCIDR = value(oc.Properties.NetworkProfile.PodCidr)
	out.Properties.NetworkProfile.ServiceCIDR = value(oc.Properties.NetworkProfile.ServiceCidr)
	out.Properties.NetworkProfile.OutboundType = api.OutboundType(value(oc.Properties.NetworkProfile.OutboundType))
	out.Properties.NetworkProfile.PreconfiguredNSG = api.PreconfiguredNSG(value(oc.Properties.NetworkProfile.PreconfiguredNSG))

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
				Count: int(value(oc.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count)),
			}
		}
		if oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs != nil {
			out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = make([]api.EffectiveOutboundIP, len(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs))
			for i := range oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs {
				if oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs[i] != nil {
					out.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs[i].ID = value(oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs[i].ID)
				}
			}
		}
	}

	out.Properties.MasterProfile.VMSize = api.VMSize(value(oc.Properties.MasterProfile.VMSize))
	out.Properties.MasterProfile.SubnetID = value(oc.Properties.MasterProfile.SubnetID)
	out.Properties.MasterProfile.EncryptionAtHost = api.EncryptionAtHost(value(oc.Properties.MasterProfile.EncryptionAtHost))
	out.Properties.MasterProfile.DiskEncryptionSetID = value(oc.Properties.MasterProfile.DiskEncryptionSetID)
	out.Properties.WorkerProfiles = nil
	if oc.Properties.WorkerProfiles != nil {
		out.Properties.WorkerProfiles = make([]api.WorkerProfile, len(oc.Properties.WorkerProfiles))
		for i := range oc.Properties.WorkerProfiles {
			if oc.Properties.WorkerProfiles[i] == nil {
				continue
			}
			out.Properties.WorkerProfiles[i].Name = value(oc.Properties.WorkerProfiles[i].Name)
			out.Properties.WorkerProfiles[i].VMSize = api.VMSize(value(oc.Properties.WorkerProfiles[i].VMSize))
			out.Properties.WorkerProfiles[i].DiskSizeGB = int(value(oc.Properties.WorkerProfiles[i].DiskSizeGB))
			out.Properties.WorkerProfiles[i].SubnetID = value(oc.Properties.WorkerProfiles[i].SubnetID)
			out.Properties.WorkerProfiles[i].Count = int(value(oc.Properties.WorkerProfiles[i].Count))
			out.Properties.WorkerProfiles[i].EncryptionAtHost = api.EncryptionAtHost(value(oc.Properties.WorkerProfiles[i].EncryptionAtHost))
			out.Properties.WorkerProfiles[i].DiskEncryptionSetID = value(oc.Properties.WorkerProfiles[i].DiskEncryptionSetID)
		}
	}
	out.Properties.WorkerProfilesStatus = nil
	if oc.Properties.WorkerProfilesStatus != nil {
		out.Properties.WorkerProfilesStatus = make([]api.WorkerProfile, len(oc.Properties.WorkerProfilesStatus))
		for i := range oc.Properties.WorkerProfilesStatus {
			if oc.Properties.WorkerProfilesStatus[i] == nil {
				continue
			}
			out.Properties.WorkerProfilesStatus[i].Name = value(oc.Properties.WorkerProfilesStatus[i].Name)
			out.Properties.WorkerProfilesStatus[i].VMSize = api.VMSize(value(oc.Properties.WorkerProfilesStatus[i].VMSize))
			out.Properties.WorkerProfilesStatus[i].DiskSizeGB = int(value(oc.Properties.WorkerProfilesStatus[i].DiskSizeGB))
			out.Properties.WorkerProfilesStatus[i].SubnetID = value(oc.Properties.WorkerProfilesStatus[i].SubnetID)
			out.Properties.WorkerProfilesStatus[i].Count = int(value(oc.Properties.WorkerProfilesStatus[i].Count))
			out.Properties.WorkerProfilesStatus[i].EncryptionAtHost = api.EncryptionAtHost(value(oc.Properties.WorkerProfilesStatus[i].EncryptionAtHost))
			out.Properties.WorkerProfilesStatus[i].DiskEncryptionSetID = value(oc.Properties.WorkerProfilesStatus[i].DiskEncryptionSetID)
		}
	}
	out.Properties.APIServerProfile.Visibility = api.Visibility(value(oc.Properties.ApiserverProfile.Visibility))
	if value(oc.Properties.ApiserverProfile.URL) != "" {
		out.Properties.APIServerProfile.URL = value(oc.Properties.ApiserverProfile.URL)
	}
	if value(oc.Properties.ApiserverProfile.IP) != "" {
		out.Properties.APIServerProfile.IP = value(oc.Properties.ApiserverProfile.IP)
	}
	out.Properties.IngressProfiles = nil
	if oc.Properties.IngressProfiles != nil {
		out.Properties.IngressProfiles = make([]api.IngressProfile, len(oc.Properties.IngressProfiles))
		for i := range oc.Properties.IngressProfiles {
			if oc.Properties.IngressProfiles[i] == nil {
				continue
			}
			out.Properties.IngressProfiles[i].Name = value(oc.Properties.IngressProfiles[i].Name)
			out.Properties.IngressProfiles[i].Visibility = api.Visibility(value(oc.Properties.IngressProfiles[i].Visibility))
			if value(oc.Properties.IngressProfiles[i].IP) != "" {
				out.Properties.IngressProfiles[i].IP = value(oc.Properties.IngressProfiles[i].IP)
			}
		}
	}

	if oc.SystemData != nil {
		out.SystemData = api.SystemData{
			CreatedBy:          value(oc.SystemData.CreatedBy),
			CreatedAt:          oc.SystemData.CreatedAt,
			CreatedByType:      api.CreatedByType(value(oc.SystemData.CreatedByType)),
			LastModifiedBy:     value(oc.SystemData.LastModifiedBy),
			LastModifiedAt:     oc.SystemData.LastModifiedAt,
			LastModifiedByType: api.CreatedByType(value(oc.SystemData.LastModifiedByType)),
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
	if oc.Properties.ConsoleProfile != nil {
		oc.Properties.ConsoleProfile.URL = nil
	}
	oc.Properties.ApiserverProfile.URL = nil
	oc.Properties.ApiserverProfile.IP = nil
	for i := range oc.Properties.IngressProfiles {
		if oc.Properties.IngressProfiles[i] != nil {
			oc.Properties.IngressProfiles[i].IP = nil
		}
	}
	oc.Properties.ClusterProfile.OidcIssuer = nil
	if oc.Properties.PlatformWorkloadIdentityProfile != nil {
		for i := range oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			if entry, ok := oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[i]; ok {
				if entry == nil {
					continue
				}
				entry.ClientID = nil
				entry.ObjectID = nil
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[i] = entry
			}
		}
	}
	if oc.Identity != nil {
		oc.Identity.PrincipalID = nil
		oc.Identity.TenantID = nil
		for i := range oc.Identity.UserAssignedIdentities {
			if entry, ok := oc.Identity.UserAssignedIdentities[i]; ok {
				if entry == nil {
					continue
				}
				entry.ClientID = nil
				entry.PrincipalID = nil
				oc.Identity.UserAssignedIdentities[i] = entry
			}
		}
	}
}
