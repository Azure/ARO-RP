package baremetal

import (
	"fmt"

	machineapi "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/openshift/installer/pkg/types"
)

// HostSettings hold the information needed to build the manifests to
// register hosts with the cluster.
type HostSettings struct {

	// Secrets holds the credential information for communicating with
	// the management controllers on the hosts.
	Secrets []corev1.Secret
}

// Hosts returns the HostSettings with details of the hardware being
// used to construct the cluster.
func Hosts(config *types.InstallConfig, machines []machineapi.Machine) (*HostSettings, error) {
	settings := &HostSettings{}

	if config.Platform.BareMetal == nil {
		return nil, fmt.Errorf("no baremetal platform in configuration")
	}

	return settings, nil
}
