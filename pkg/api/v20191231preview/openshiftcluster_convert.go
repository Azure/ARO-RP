package v20191231preview

import (
	"github.com/jim-minter/rp/pkg/api"
)

// openShiftClusterToExternal returns a new external representation of the
// internal object, reading from the subset of the internal object's fields that
// appear in the external representation.  ToExternal does not modify its
// argument; there is no pointer aliasing between the passed and returned
// objects
func openShiftClusterToExternal(oc *api.OpenShiftCluster) *OpenShiftCluster {
	out := &OpenShiftCluster{
		ID:       oc.ID,
		Name:     oc.Name,
		Type:     oc.Type,
		Location: oc.Location,
		Properties: Properties{
			ProvisioningState: ProvisioningState(oc.Properties.ProvisioningState),
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
			APIServerURL: oc.Properties.APIServerURL,
			ConsoleURL:   oc.Properties.ConsoleURL,
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

	if oc.Tags != nil {
		out.Tags = make(map[string]string, len(oc.Tags))
		for k, v := range oc.Tags {
			out.Tags[k] = v
		}
	}

	return out
}

// openShiftClustersToExternal returns a slice of external representations of
// the internal objects
func openShiftClustersToExternal(ocs []*api.OpenShiftCluster) *OpenShiftClusterList {
	l := &OpenShiftClusterList{
		OpenShiftClusters: make([]*OpenShiftCluster, 0, len(ocs)),
	}

	for _, oc := range ocs {
		l.OpenShiftClusters = append(l.OpenShiftClusters, openShiftClusterToExternal(oc))
	}

	return l
}

// openShiftClusterToInternal overwrites in place a pre-existing internal
// object, setting (only) all mapped fields from the external representation.
// ToInternal modifies its argument; there is no pointer aliasing between the
// passed and returned objects
func openShiftClusterToInternal(oc *OpenShiftCluster, out *api.OpenShiftCluster) {
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
	out.Properties.APIServerURL = oc.Properties.APIServerURL
	out.Properties.ConsoleURL = oc.Properties.ConsoleURL
}
