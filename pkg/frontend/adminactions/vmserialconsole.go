package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (a *azureActions) VMSerialConsole(ctx context.Context, w http.ResponseWriter,
	log *logrus.Entry, vmName string) error {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	blob, err := a.virtualMachines.GetSerialConsoleForVM(ctx, clusterRGName, vmName)
	if err != nil {
		return err
	}

	w.Header().Add("Content-Type", "text/plain")

	_, err = io.Copy(w, blob)
	return err
}
