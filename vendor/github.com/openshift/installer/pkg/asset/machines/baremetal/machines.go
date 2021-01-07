// Package baremetal generates Machine objects for bare metal.
package baremetal

import (
	machineapi "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"

	"github.com/openshift/installer/pkg/types"
)

// Machines returns a list of machines for a machinepool.
func Machines(clusterID string, config *types.InstallConfig, pool *types.MachinePool, osImage, role, userDataSecret string) ([]machineapi.Machine, error) {
	var machines []machineapi.Machine
	return machines, nil
}
