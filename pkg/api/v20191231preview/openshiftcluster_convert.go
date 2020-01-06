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
	for _, p := range oc.Properties.WorkerProfiles {
		var outp *api.WorkerProfile
		for i, pp := range out.Properties.WorkerProfiles {
			if pp.Name == p.Name {
				outp = &out.Properties.WorkerProfiles[i]
				break
			}
		}
		if outp == nil {
			out.Properties.WorkerProfiles = append(out.Properties.WorkerProfiles, api.WorkerProfile{})
			outp = &out.Properties.WorkerProfiles[len(out.Properties.WorkerProfiles)-1]
		}
		outp.Name = p.Name
		outp.VMSize = api.VMSize(p.VMSize)
		outp.DiskSizeGB = p.DiskSizeGB
		outp.SubnetID = p.SubnetID
		outp.Count = p.Count
	}
	out.Properties.APIServerProfile.Visibility = api.Visibility(oc.Properties.APIServerProfile.Visibility)
	out.Properties.APIServerProfile.URL = oc.Properties.APIServerProfile.URL
	out.Properties.APIServerProfile.IP = oc.Properties.APIServerProfile.IP
	for _, p := range oc.Properties.IngressProfiles {
		var outp *api.IngressProfile
		for i, pp := range out.Properties.IngressProfiles {
			if pp.Name == p.Name {
				outp = &out.Properties.IngressProfiles[i]
				break
			}
		}
		if outp == nil {
			out.Properties.IngressProfiles = append(out.Properties.IngressProfiles, api.IngressProfile{})
			outp = &out.Properties.IngressProfiles[len(out.Properties.IngressProfiles)-1]
		}
		outp.Name = p.Name
		outp.Visibility = api.Visibility(p.Visibility)
		outp.IP = p.IP
	}
	out.Properties.ConsoleURL = oc.Properties.ConsoleURL
}
