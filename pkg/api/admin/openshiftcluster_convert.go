package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type openShiftClusterConverter struct{}

// ToExternal returns a new external representation of the internal object,
// reading from the subset of the internal object's fields that appear in the
// external representation.  ToExternal does not modify its argument; there is
// no pointer aliasing between the passed and returned objects
func (c *openShiftClusterConverter) ToExternal(oc *api.OpenShiftCluster) interface{} {
	out := &OpenShiftCluster{
		ID:       oc.ID,
		Name:     oc.Name,
		Type:     oc.Type,
		Location: oc.Location,
		Properties: OpenShiftClusterProperties{
			ArchitectureVersion:     ArchitectureVersion(oc.Properties.ArchitectureVersion),
			ProvisioningState:       ProvisioningState(oc.Properties.ProvisioningState),
			LastProvisioningState:   ProvisioningState(oc.Properties.LastProvisioningState),
			FailedProvisioningState: ProvisioningState(oc.Properties.FailedProvisioningState),
			LastAdminUpdateError:    oc.Properties.LastAdminUpdateError,
			CreatedBy:               oc.Properties.CreatedBy,
			ProvisionedBy:           oc.Properties.ProvisionedBy,
			ClusterProfile: ClusterProfile{
				Domain:          oc.Properties.ClusterProfile.Domain,
				Version:         oc.Properties.ClusterProfile.Version,
				ResourceGroupID: oc.Properties.ClusterProfile.ResourceGroupID,
			},
			ConsoleProfile: ConsoleProfile{
				URL: oc.Properties.ConsoleProfile.URL,
			},
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientID: oc.Properties.ServicePrincipalProfile.ClientID,
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:           oc.Properties.NetworkProfile.PodCIDR,
				ServiceCIDR:       oc.Properties.NetworkProfile.ServiceCIDR,
				PrivateEndpointIP: oc.Properties.NetworkProfile.PrivateEndpointIP,
			},
			MasterProfile: MasterProfile{
				VMSize:   VMSize(oc.Properties.MasterProfile.VMSize),
				SubnetID: oc.Properties.MasterProfile.SubnetID,
			},
			APIServerProfile: APIServerProfile{
				Visibility: Visibility(oc.Properties.APIServerProfile.Visibility),
				URL:        oc.Properties.APIServerProfile.URL,
				IP:         oc.Properties.APIServerProfile.IP,
			},
			StorageSuffix: oc.Properties.StorageSuffix,
		},
	}

	if oc.Properties.WorkerProfiles != nil {
		out.Properties.WorkerProfiles = make([]WorkerProfile, 0, len(oc.Properties.WorkerProfiles))
		for _, p := range oc.Properties.WorkerProfiles {
			out.Properties.WorkerProfiles = append(out.Properties.WorkerProfiles, WorkerProfile{
				Name:       p.Name,
				VMSize:     VMSize(p.VMSize),
				DiskSizeGB: p.DiskSizeGB,
				SubnetID:   p.SubnetID,
				Count:      p.Count,
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

	if oc.Properties.Install != nil {
		out.Properties.Install = &Install{
			Now:   oc.Properties.Install.Now,
			Phase: InstallPhase(oc.Properties.Install.Phase),
		}
	}

	if oc.Tags != nil {
		out.Tags = make(map[string]string, len(oc.Tags))
		for k, v := range oc.Tags {
			out.Tags[k] = v
		}
	}

	if oc.Properties.RegistryProfiles != nil {
		out.Properties.RegistryProfiles = make([]RegistryProfile, len(oc.Properties.RegistryProfiles))
		for i, v := range oc.Properties.RegistryProfiles {
			out.Properties.RegistryProfiles[i].Name = v.Name
			out.Properties.RegistryProfiles[i].Username = v.Username
		}
	}

	return out
}

// ToExternalList returns a slice of external representations of the internal
// objects
func (c *openShiftClusterConverter) ToExternalList(ocs []*api.OpenShiftCluster, nextLink string) interface{} {
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
func (c *openShiftClusterConverter) ToInternal(_oc interface{}, out *api.OpenShiftCluster) {
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
	out.Properties.ArchitectureVersion = api.ArchitectureVersion(oc.Properties.ArchitectureVersion)
	out.Properties.ProvisioningState = api.ProvisioningState(oc.Properties.ProvisioningState)
	out.Properties.LastProvisioningState = api.ProvisioningState(oc.Properties.LastProvisioningState)
	out.Properties.FailedProvisioningState = api.ProvisioningState(oc.Properties.FailedProvisioningState)
	out.Properties.LastAdminUpdateError = oc.Properties.LastAdminUpdateError
	out.Properties.CreatedBy = oc.Properties.CreatedBy
	out.Properties.ProvisionedBy = oc.Properties.ProvisionedBy
	out.Properties.ClusterProfile.Domain = oc.Properties.ClusterProfile.Domain
	out.Properties.ClusterProfile.Version = oc.Properties.ClusterProfile.Version
	out.Properties.ClusterProfile.ResourceGroupID = oc.Properties.ClusterProfile.ResourceGroupID
	out.Properties.ConsoleProfile.URL = oc.Properties.ConsoleProfile.URL
	out.Properties.ServicePrincipalProfile.ClientID = oc.Properties.ServicePrincipalProfile.ClientID
	out.Properties.NetworkProfile.PodCIDR = oc.Properties.NetworkProfile.PodCIDR
	out.Properties.NetworkProfile.ServiceCIDR = oc.Properties.NetworkProfile.ServiceCIDR
	out.Properties.NetworkProfile.PrivateEndpointIP = oc.Properties.NetworkProfile.PrivateEndpointIP
	out.Properties.MasterProfile.VMSize = api.VMSize(oc.Properties.MasterProfile.VMSize)
	out.Properties.MasterProfile.SubnetID = oc.Properties.MasterProfile.SubnetID
	out.Properties.StorageSuffix = oc.Properties.StorageSuffix
	out.Properties.WorkerProfiles = nil
	if oc.Properties.WorkerProfiles != nil {
		out.Properties.WorkerProfiles = make([]api.WorkerProfile, len(oc.Properties.WorkerProfiles))
		for i := range oc.Properties.WorkerProfiles {
			out.Properties.WorkerProfiles[i].Name = oc.Properties.WorkerProfiles[i].Name
			out.Properties.WorkerProfiles[i].VMSize = api.VMSize(oc.Properties.WorkerProfiles[i].VMSize)
			out.Properties.WorkerProfiles[i].DiskSizeGB = oc.Properties.WorkerProfiles[i].DiskSizeGB
			out.Properties.WorkerProfiles[i].SubnetID = oc.Properties.WorkerProfiles[i].SubnetID
			out.Properties.WorkerProfiles[i].Count = oc.Properties.WorkerProfiles[i].Count
		}
	}
	out.Properties.APIServerProfile.Visibility = api.Visibility(oc.Properties.APIServerProfile.Visibility)
	out.Properties.APIServerProfile.URL = oc.Properties.APIServerProfile.URL
	out.Properties.APIServerProfile.IP = oc.Properties.APIServerProfile.IP
	out.Properties.IngressProfiles = nil
	if oc.Properties.IngressProfiles != nil {
		out.Properties.IngressProfiles = make([]api.IngressProfile, len(oc.Properties.IngressProfiles))
		for i := range oc.Properties.IngressProfiles {
			out.Properties.IngressProfiles[i].Name = oc.Properties.IngressProfiles[i].Name
			out.Properties.IngressProfiles[i].Visibility = api.Visibility(oc.Properties.IngressProfiles[i].Visibility)
			out.Properties.IngressProfiles[i].IP = oc.Properties.IngressProfiles[i].IP
		}
	}

	out.Properties.Install = nil
	if oc.Properties.Install != nil {
		out.Properties.Install = &api.Install{
			Now:   oc.Properties.Install.Now,
			Phase: api.InstallPhase(oc.Properties.Install.Phase),
		}
	}

	// out.Properties.RegistryProfiles is not converted. The field is immutable and does not have to be converted.
	// Other fields are converted and this breaks the pattern, however this converting this field creates an issue
	// with filling the out.Properties.RegistryProfiles[i].Password as default is "" which erases the original value.
	// Workaround would be filling the password when receiving request, but it is array and the logic would be to complex.
}
