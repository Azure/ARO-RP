// Package baremetal generates Machine objects for bare metal.
package baremetal

import (
	machineapi "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"

	"github.com/openshift/installer/pkg/types"
)

// MachineSets returns a list of machinesets for a machinepool.
func MachineSets(clusterID string, config *types.InstallConfig, pool *types.MachinePool, osImage, role, userDataSecret string) ([]*machineapi.MachineSet, error) {
	return []*machineapi.MachineSet{}, nil
}
