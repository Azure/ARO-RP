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
func apiServersReady(context.Context) (bool, error)                        { return false, nil }
func minimumWorkerNodesReady(context.Context) (bool, error)                { return false, nil }
func operatorConsoleExists(context.Context) (bool, error)                  { return false, nil }
func operatorConsoleReady(context.Context) (bool, error)                   { return false, nil }
func clusterVersionReady(context.Context) (bool, error)                    { return false, nil }
func ingressControllerReady(context.Context) (bool, error)                 { return false, nil }
func aroDeploymentReady(context.Context) (bool, error)                     { return false, nil }
func ensureAROOperatorRunningDesiredVersion(context.Context) (bool, error) { return false, nil }
func hiveClusterDeploymentReady(context.Context) (bool, error)             { return false, nil }
func hiveClusterInstallationComplete(context.Context) (bool, error)        { return false, nil }

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
			desc:     "test conditionfail for func - apiServersReady",
			function: apiServersReady,
			wantErr:  "500: DeploymentFailed: : Kube API has not initialised successfully and is unavailable.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - minimumWorkerNodesReady",
			function: minimumWorkerNodesReady,
			wantErr:  "500: DeploymentFailed: : Minimum number of worker nodes have not been successfully created.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - operatorConsoleExists",
			function: operatorConsoleExists,
			wantErr:  "500: DeploymentFailed: : Console Cluster Operator has failed to initialize successfully.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - operatorConsoleReady",
			function: operatorConsoleReady,
			wantErr:  "500: DeploymentFailed: : Console Cluster Operator has not started successfully.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - clusterVersionReady",
			function: clusterVersionReady,
			wantErr:  "500: DeploymentFailed: : Cluster Verion is not reporting status as ready.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - clusterVersionReady",
			function: ingressControllerReady,
			wantErr:  "500: DeploymentFailed: : Ingress Cluster Operator has not started successfully.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - aroDeploymentReady",
			function: aroDeploymentReady,
			wantErr:  "500: DeploymentFailed: : ARO Cluster Operator has failed to initialize successfully.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - ensureAROOperatorRunningDesiredVersion",
			function: ensureAROOperatorRunningDesiredVersion,
			wantErr:  "500: DeploymentFailed: : ARO Cluster Operator is not running desired version.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - hiveClusterDeploymentReady",
			function: hiveClusterDeploymentReady,
			wantErr:  "500: DeploymentFailed: : Timed out waiting for a condition, cluster Installation is unsuccessful.Please retry, if issue persists: raise azure support ticket",
		},
		{
			desc:     "test conditionfail for func - hiveClusterInstallationComplete",
			function: hiveClusterInstallationComplete,
			wantErr:  "500: DeploymentFailed: : Timed out waiting for a condition, cluster Installation is unsuccessful.Please retry, if issue persists: raise azure support ticket",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := enrichConditionTimeoutError(tt.function, errors.New(tt.originalErr)); got.Error() != tt.wantErr {
				t.Errorf("invlaid enrichConditionTimeoutError: %s, got: %s", tt.wantErr, got)
			}
		})
	}
}
