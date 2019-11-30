package installconfig

import (
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig/azure"
)

type PlatformCreds struct {
	Azure *azure.Credentials
}

var _ asset.Asset = (*PlatformCreds)(nil)

func (a *PlatformCreds) Dependencies() []asset.Asset {
	return nil
}

func (a *PlatformCreds) Generate(dependencies asset.Parents) error {
	return nil
}

func (a *PlatformCreds) Name() string {
	return "Platform Credentials"
}
