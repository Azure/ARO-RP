package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
)

func (i *Installer) deploy(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image) error {
	err := i.installStorage(ctx, installConfig, platformCreds, image)
	if err != nil {
		return err
	}

	return i.installResources(ctx)
}
