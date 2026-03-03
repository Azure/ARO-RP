package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (a *azureActions) VMSerialConsole(ctx context.Context,
	log *logrus.Entry, vmName string, target io.Writer,
) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	return a.virtualMachines.GetSerialConsoleForVM(ctx, clusterRGName, vmName, target)
}
