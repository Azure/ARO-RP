package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
)

// functionnames that will be used in the conditionFunction below
// All the keys of map timeoutConditionErrors
func attachNSGs(context.Context) (bool, bool, error)              { return false, false, nil }
func apiServersReady(context.Context) (bool, bool, error)         { return false, false, nil }
func minimumWorkerNodesReady(context.Context) (bool, bool, error) { return false, false, nil }
func operatorConsoleExists(context.Context) (bool, bool, error)   { return false, false, nil }
func operatorConsoleReady(context.Context) (bool, bool, error)    { return false, false, nil }
func clusterVersionReady(context.Context) (bool, bool, error)     { return false, false, nil }
func ingressControllerReady(context.Context) (bool, bool, error)  { return false, false, nil }
func aroDeploymentReady(context.Context) (bool, bool, error)      { return false, false, nil }
func ensureAROOperatorRunningDesiredVersion(context.Context) (bool, bool, error) {
	return false, false, nil
}
func hiveClusterDeploymentReady(context.Context) (bool, bool, error)      { return false, false, nil }
func hiveClusterInstallationComplete(context.Context) (bool, bool, error) { return false, false, nil }

func TestEnrichConditionTimeoutError(t *testing.T) {
	for _, tt := range []struct {
		desc        string
		function    conditionFunction
		originalErr string
		wantErr     string
	}{
		// Verify response for func's mention in timeoutConditionErrors and
		// Emit generic Error if an unknown func
		{
			// unknown function
			desc:        "test conditionfail for func - unknownFunc",
			function:    timingOutCondition,
			originalErr: "timed out waiting for the condition",
			wantErr:     "timed out waiting for the condition",
		},
		{
			desc:     "test conditionfail for func - attachNSGs",
			function: attachNSGs,
			wantErr:  "500: DeploymentFailed: : Failed to attach the ARO NSG to the cluster subnets. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - apiServersReady",
			function: apiServersReady,
			wantErr:  "500: DeploymentFailed: : Kube API has not initialised successfully and is unavailable. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - minimumWorkerNodesReady",
			function: minimumWorkerNodesReady,
			wantErr:  "500: DeploymentFailed: : Minimum number of worker nodes have not been successfully created. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - operatorConsoleExists",
			function: operatorConsoleExists,
			wantErr:  "500: DeploymentFailed: : Console Cluster Operator has failed to initialize successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - operatorConsoleReady",
			function: operatorConsoleReady,
			wantErr:  "500: DeploymentFailed: : Console Cluster Operator has not started successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - clusterVersionReady",
			function: clusterVersionReady,
			wantErr:  "500: DeploymentFailed: : Cluster Verion is not reporting status as ready. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - clusterVersionReady",
			function: ingressControllerReady,
			wantErr:  "500: DeploymentFailed: : Ingress Cluster Operator has not started successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - aroDeploymentReady",
			function: aroDeploymentReady,
			wantErr:  "500: DeploymentFailed: : ARO Cluster Operator has failed to initialize successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - ensureAROOperatorRunningDesiredVersion",
			function: ensureAROOperatorRunningDesiredVersion,
			wantErr:  "500: DeploymentFailed: : ARO Cluster Operator is not running desired version. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - hiveClusterDeploymentReady",
			function: hiveClusterDeploymentReady,
			wantErr:  "500: DeploymentFailed: : Timed out waiting for the condition to be ready. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - hiveClusterInstallationComplete",
			function: hiveClusterInstallationComplete,
			wantErr:  "500: DeploymentFailed: : Timed out waiting for the condition to complete. Please retry, and if the issue persists, raise an Azure support ticket",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := enrichConditionTimeoutError(tt.function, errors.New(tt.originalErr)); got.Error() != tt.wantErr {
				t.Errorf("invalid enrichConditionTimeoutError: %s, got: %s", tt.wantErr, got)
			}
		})
	}
}
