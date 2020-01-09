package v20191231preview

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
		Properties: Properties{
			ProvisioningState: ProvisioningState(oc.Properties.ProvisioningState),
			ClusterDomain:     oc.Properties.ClusterDomain,
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientID:     oc.Properties.ServicePrincipalProfile.ClientID,
				ClientSecret: oc.Properties.ServicePrincipalProfile.ClientSecret,
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:     oc.Properties.NetworkProfile.PodCIDR,
				ServiceCIDR: oc.Properties.NetworkProfile.ServiceCIDR,
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
			ConsoleURL: oc.Properties.ConsoleURL,
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

	if oc.Tags != nil {
		out.Tags = make(map[string]string, len(oc.Tags))
		for k, v := range oc.Tags {
			out.Tags[k] = v
		}
	}

	return out
}

// ToExternalList returns a slice of external representations of the internal
// objects
func (c *openShiftClusterConverter) ToExternalList(ocs []*api.OpenShiftCluster) interface{} {
	l := &OpenShiftClusterList{
		OpenShiftClusters: make([]*OpenShiftCluster, 0, len(ocs)),
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
	out.Properties.ProvisioningState = api.ProvisioningState(oc.Properties.ProvisioningState)
	out.Properties.ClusterDomain = oc.Properties.ClusterDomain
	out.Properties.ServicePrincipalProfile.ClientID = oc.Properties.ServicePrincipalProfile.ClientID
	out.Properties.ServicePrincipalProfile.ClientSecret = oc.Properties.ServicePrincipalProfile.ClientSecret
	out.Properties.NetworkProfile.PodCIDR = oc.Properties.NetworkProfile.PodCIDR
	out.Properties.NetworkProfile.ServiceCIDR = oc.Properties.NetworkProfile.ServiceCIDR
	out.Properties.MasterProfile.VMSize = api.VMSize(oc.Properties.MasterProfile.VMSize)
	out.Properties.MasterProfile.SubnetID = oc.Properties.MasterProfile.SubnetID
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
	out.Properties.ConsoleURL = oc.Properties.ConsoleURL
}
