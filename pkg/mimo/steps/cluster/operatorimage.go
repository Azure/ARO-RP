package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func UpdateAROOperatorImage(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	deployer, err := th.AROOperatorDeployer()
	if err != nil {
		return mimo.TerminalError(fmt.Errorf("failed to create ARO operator deployer: %w", err))
	}

	err = deployer.Update(ctx)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("failed to update ARO operator image: %w", err))
	}

	return nil
}

func AROOperatorDeploymentReady(ctx context.Context) (bool, error) {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return false, mimo.TerminalError(err)
	}

	deployer, err := th.AROOperatorDeployer()
	if err != nil {
		return false, mimo.TerminalError(fmt.Errorf("failed to create ARO operator deployer: %w", err))
	}

	ok, err := deployer.IsReady(ctx)
	if err != nil {
		return false, mimo.TransientError(fmt.Errorf("failed to check ARO operator deployment readiness: %w", err))
	}

	return ok, nil
}

func EnsureAROOperatorRunningDesiredVersion(ctx context.Context) (bool, error) {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return false, mimo.TerminalError(err)
	}

	deployer, err := th.AROOperatorDeployer()
	if err != nil {
		return false, mimo.TerminalError(fmt.Errorf("failed to create ARO operator deployer: %w", err))
	}

	ok, err := deployer.IsRunningDesiredVersion(ctx)
	if err != nil {
		return false, mimo.TransientError(fmt.Errorf("failed to check ARO operator image version: %w", err))
	}

	return ok, nil
}

func SyncClusterObject(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	deployer, err := th.AROOperatorDeployer()
	if err != nil {
		return mimo.TerminalError(fmt.Errorf("failed to create ARO operator deployer: %w", err))
	}

	err = deployer.SyncClusterObject(ctx)
	if err != nil {
		return mimo.TransientError(fmt.Errorf("failed to sync ARO cluster object: %w", err))
	}

	th.SetResultMessage(fmt.Sprintf("updated ARO operator image to %s", th.Environment().AROOperatorImage()))
	return nil
}
