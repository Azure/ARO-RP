package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
)

type teststruct struct{}

// functionnames that will be used in the conditionFunction below
// All the keys of map timeoutConditionErrors
func (n *teststruct) attachNSGs(context.Context) (bool, error)              { return false, nil }
func (n *teststruct) apiServersReady(context.Context) (bool, error)         { return false, nil }
func (n *teststruct) minimumWorkerNodesReady(context.Context) (bool, error) { return false, nil }
func (n *teststruct) operatorConsoleExists(context.Context) (bool, error)   { return false, nil }
func (n *teststruct) operatorConsoleReady(context.Context) (bool, error)    { return false, nil }
func (n *teststruct) clusterVersionReady(context.Context) (bool, error)     { return false, nil }
func (n *teststruct) ingressControllerReady(context.Context) (bool, error)  { return false, nil }
func (n *teststruct) aroDeploymentReady(context.Context) (bool, error)      { return false, nil }
func (n *teststruct) ensureAROOperatorRunningDesiredVersion(context.Context) (bool, error) {
	return false, nil
}
func (n *teststruct) hiveClusterDeploymentReady(context.Context) (bool, error) { return false, nil }
func (n *teststruct) hiveClusterInstallationComplete(context.Context) (bool, error) {
	return false, nil
}

func TestEnrichConditionTimeoutError(t *testing.T) {
	// When stringifying a method on a struct, golang adds -fm -- this is not
	// useful to us, so using a struct instance here will verify that it is not
	// present when matching the timeout error strings
	s := &teststruct{}

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
			function: s.attachNSGs,
			wantErr:  "500: DeploymentFailed: : Failed to attach the ARO NSG to the cluster subnets. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - apiServersReady",
			function: s.apiServersReady,
			wantErr:  "500: DeploymentFailed: : Kube API has not initialised successfully and is unavailable. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - minimumWorkerNodesReady",
			function: s.minimumWorkerNodesReady,
			wantErr:  "500: DeploymentFailed: : Minimum number of worker nodes have not been successfully created. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - operatorConsoleExists",
			function: s.operatorConsoleExists,
			wantErr:  "500: DeploymentFailed: : Console Cluster Operator has failed to initialize successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - operatorConsoleReady",
			function: s.operatorConsoleReady,
			wantErr:  "500: DeploymentFailed: : Console Cluster Operator has not started successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - clusterVersionReady",
			function: s.clusterVersionReady,
			wantErr:  "500: DeploymentFailed: : Cluster Version is not reporting status as ready. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - clusterVersionReady",
			function: s.ingressControllerReady,
			wantErr:  "500: DeploymentFailed: : Ingress Cluster Operator has not started successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - aroDeploymentReady",
			function: s.aroDeploymentReady,
			wantErr:  "500: DeploymentFailed: : ARO Cluster Operator has failed to initialize successfully. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - ensureAROOperatorRunningDesiredVersion",
			function: s.ensureAROOperatorRunningDesiredVersion,
			wantErr:  "500: DeploymentFailed: : ARO Cluster Operator is not running desired version. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - hiveClusterDeploymentReady",
			function: s.hiveClusterDeploymentReady,
			wantErr:  "500: DeploymentFailed: : Timed out waiting for the condition to be ready. Please retry, and if the issue persists, raise an Azure support ticket",
		},
		{
			desc:     "test conditionfail for func - hiveClusterInstallationComplete",
			function: s.hiveClusterInstallationComplete,
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
