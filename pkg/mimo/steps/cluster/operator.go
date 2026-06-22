package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

// Reset the Operator Version setting in CosmosDB to blank (= Operator version
// matches version of RP deploying). Does not directly update the Operator in-cluster.
func ResetOperatorVersion(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	_, err = th.PatchOpenShiftClusterDocument(ctx, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.OperatorVersion = ""
		return nil
	})
	if err != nil {
		if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
			return mimo.TerminalError(err)
		}
		return mimo.TransientError(err)
	}
	return nil
}
