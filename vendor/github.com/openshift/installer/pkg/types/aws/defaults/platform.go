package defaults

import (
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/aws"
)

var (
	defaultMachineClass = map[types.Architecture]map[string][]string{
		types.ArchitectureAMD64: {
			// Example region default machine class override for AMD64:
			// "ap-east-1":      {"m5", "m4"},
		},
		types.ArchitectureARM64: {
			// Example region default machine class override for ARM64:
			// "us-east-1":      {"m6g", "m6gd"},
		},
	}
)

// SetPlatformDefaults sets the defaults for the platform.
func SetPlatformDefaults(p *aws.Platform) {
}

// InstanceClasses returns a list of instance "class", in decreasing priority order, which we should use for a given
// region. Default is m6i then m5 unless a region override is defined in defaultMachineClass.
func InstanceClasses(region string, arch types.Architecture) []string {
	if classesForArch, ok := defaultMachineClass[arch]; ok {
		if classes, ok := classesForArch[region]; ok {
			return classes
		}
	}

	switch arch {
	case types.ArchitectureARM64:
		return []string{"m6g"}
	default:
		return []string{"m6i", "m5"}
	}
}
