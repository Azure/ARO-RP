package installconfig

import (
	"fmt"

	"github.com/openshift/installer/pkg/asset"
	azconfig "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/types/azure"
)

// PlatformProvisionCheck is an asset that validates the install-config platform for
// any requirements specific for provisioning infrastructure.
type PlatformProvisionCheck struct {
}

var _ asset.Asset = (*PlatformProvisionCheck)(nil)

// Dependencies returns the dependencies for PlatformProvisionCheck
func (a *PlatformProvisionCheck) Dependencies() []asset.Asset {
	return []asset.Asset{
		&InstallConfig{},
	}
}

// Generate queries for input from the user.
func (a *PlatformProvisionCheck) Generate(dependencies asset.Parents) error {
	ic := &InstallConfig{}
	platformCreds := &PlatformCreds{}
	dependencies.Get(ic)
	dependencies.Get(platformCreds)

	var err error
	platform := ic.Config.Platform.Name()
	switch platform {
	case azure.Name:
		err = azconfig.ValidatePublicDNS(platformCreds.Azure, ic.Config)
		if err != nil {
			return err
		}
	default:
		err = fmt.Errorf("unknown platform type %q", platform)
	}
	return err
}

// Name returns the human-friendly name of the asset.
func (a *PlatformProvisionCheck) Name() string {
	return "Platform Provisioning Check"
}
