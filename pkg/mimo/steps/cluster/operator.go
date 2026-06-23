package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/operator/deploy"
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

func getOperatorDeployer(ctx context.Context) (deploy.Operator, error) {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return nil, mimo.TerminalError(err)
	}

	ch, err := th.ClientHelper()
	if err != nil {
		return nil, mimo.TransientError(err)
	}

	deployer, err := deploy.New(
		th.Log(),
		th.Environment(),
		th.GetOpenShiftClusterDocument().OpenShiftCluster,
		th.GetSubscriptionDocument(),
		ch,
	)
	if err != nil {
		return nil, mimo.TransientError(fmt.Errorf("failed creating operator deployer: %w", err))
	}

	return deployer, nil
}

// Deploy the Operator into the cluster, similar to how an AdminUpdate would.
// Does not sync the Cluster resource, see SyncClusterObject for that.
func DeployOperatorIntoCluster(ctx context.Context) error {
	deployer, err := getOperatorDeployer(ctx)
	if err != nil {
		return err
	}

	err = deployer.Update(ctx)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("failed updating operator: %w", err))
	}

	return nil
}

// Syncs the Cluster object in the cluster with the data stored in CosmosDB.
func SyncClusterObject(ctx context.Context) error {
	deployer, err := getOperatorDeployer(ctx)
	if err != nil {
		return err
	}

	err = deployer.SyncClusterObject(ctx)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("failed syncing cluster resource: %w", err))
	}
	return nil
}

// Check whether the ARO Operator is ready.
func WaitForAROOperatorReady(ctx context.Context) (bool, error) {
	deployer, err := getOperatorDeployer(ctx)
	if err != nil {
		return false, err
	}

	ready, err := deployer.IsReady(ctx)
	if err != nil {
		return false, mimo.TransientError(fmt.Errorf("error when checking Operator readiness: %w", err))
	}

	return ready, nil
}

// Check whether the ARO Operator is at the desired version.
func WaitForAROOperatorRunningDesiredVersion(ctx context.Context) (bool, error) {
	deployer, err := getOperatorDeployer(ctx)
	if err != nil {
		return false, err
	}

	ready, err := deployer.IsRunningDesiredVersion(ctx)
	if err != nil {
		return false, mimo.TransientError(fmt.Errorf("error when checking if Operator is desired version: %w", err))
	}

	return ready, nil
}
