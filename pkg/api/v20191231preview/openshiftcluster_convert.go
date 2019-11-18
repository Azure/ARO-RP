package v20191231preview

import (
	"github.com/jim-minter/rp/pkg/api"
)

// OpenShiftClusterToExternal returns a new external representation of the
// internal object, reading from the subset of the internal object's fields that
// appear in the external representation.  ToExternal does not modify its
// argument; there is no pointer aliasing between the passed and returned
// objects.
func OpenShiftClusterToExternal(oc *api.OpenShiftCluster) *OpenShiftCluster {
	out := &OpenShiftCluster{
		ID:       oc.ID,
		Name:     oc.Name,
		Type:     oc.Type,
		Location: oc.Location,
		Properties: Properties{
			ProvisioningState: ProvisioningState(oc.Properties.ProvisioningState),
			NetworkProfile: NetworkProfile{
				VNetCIDR:    oc.Properties.NetworkProfile.VNetCIDR,
				PodCIDR:     oc.Properties.NetworkProfile.PodCIDR,
				ServiceCIDR: oc.Properties.NetworkProfile.ServiceCIDR,
			},
			MasterProfile: MasterProfile{
				VMSize: VMSize(oc.Properties.MasterProfile.VMSize),
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

// ToInternal overwrites in place a pre-existing internal object, setting (only)
// all mapped fields from the external representation.  ToInternal modifies its
// argument; there is no pointer aliasing between the passed and returned
// objects.
func (oc *OpenShiftCluster) ToInternal(out *api.OpenShiftCluster) {
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
	out.Properties.NetworkProfile.VNetCIDR = oc.Properties.NetworkProfile.VNetCIDR
	out.Properties.NetworkProfile.PodCIDR = oc.Properties.NetworkProfile.PodCIDR
	out.Properties.NetworkProfile.ServiceCIDR = oc.Properties.NetworkProfile.ServiceCIDR
	out.Properties.MasterProfile.VMSize = api.VMSize(oc.Properties.MasterProfile.VMSize)
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
		outp.Count = p.Count
	}
	out.Properties.APIServerURL = oc.Properties.APIServerURL
	out.Properties.ConsoleURL = oc.Properties.ConsoleURL
}
