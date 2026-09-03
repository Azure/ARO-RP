package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestRunCommandResultError(t *testing.T) {
	const (
		vmScaleSetName = "rp-vmss-test"
		instanceID     = "3"
	)

	for _, tt := range []struct {
		name    string
		result  mgmtcompute.RunCommandResult
		wantErr string
	}{
		{
			name:   "no value reported",
			result: mgmtcompute.RunCommandResult{},
		},
		{
			name: "no statuses reported",
			result: mgmtcompute.RunCommandResult{
				Value: &[]mgmtcompute.InstanceViewStatus{},
			},
		},
		{
			name: "informational status is not an error",
			result: mgmtcompute.RunCommandResult{
				Value: &[]mgmtcompute.InstanceViewStatus{
					{
						Code:    pointerutils.ToPtr("ProvisioningState/succeeded"),
						Level:   mgmtcompute.Info,
						Message: pointerutils.ToPtr("Enable succeeded: \n[stdout]\n\n[stderr]\n"),
					},
				},
			},
		},
		{
			name: "warning is not an error",
			result: mgmtcompute.RunCommandResult{
				Value: &[]mgmtcompute.InstanceViewStatus{
					{
						Code:  pointerutils.ToPtr("ComponentStatus/StdErr/succeeded"),
						Level: mgmtcompute.Warning,
					},
				},
			},
		},
		{
			name: "error status is reported with its code and message",
			result: mgmtcompute.RunCommandResult{
				Value: &[]mgmtcompute.InstanceViewStatus{
					{
						Code:    pointerutils.ToPtr("ComponentStatus/StdErr/failed"),
						Level:   mgmtcompute.Error,
						Message: pointerutils.ToPtr("Failed to restart aro-portal.service: Unit not found."),
					},
				},
			},
			wantErr: "run command on scale set rp-vmss-test instance 3 reported an error: ComponentStatus/StdErr/failed: Failed to restart aro-portal.service: Unit not found.",
		},
		{
			name: "error status carrying no detail is still reported",
			result: mgmtcompute.RunCommandResult{
				Value: &[]mgmtcompute.InstanceViewStatus{
					{
						Level: mgmtcompute.Error,
					},
				},
			},
			wantErr: "run command on scale set rp-vmss-test instance 3 reported an error: no detail reported",
		},
		{
			name: "an error among informational statuses is not lost",
			result: mgmtcompute.RunCommandResult{
				Value: &[]mgmtcompute.InstanceViewStatus{
					{
						Code:  pointerutils.ToPtr("ProvisioningState/succeeded"),
						Level: mgmtcompute.Info,
					},
					{
						Code:  pointerutils.ToPtr("ComponentStatus/StdErr/failed"),
						Level: mgmtcompute.Error,
					},
				},
			},
			wantErr: "run command on scale set rp-vmss-test instance 3 reported an error: ComponentStatus/StdErr/failed",
		},
		{
			name: "every error is reported, not just the first",
			result: mgmtcompute.RunCommandResult{
				Value: &[]mgmtcompute.InstanceViewStatus{
					{
						Code:  pointerutils.ToPtr("first"),
						Level: mgmtcompute.Error,
					},
					{
						Code:  pointerutils.ToPtr("second"),
						Level: mgmtcompute.Error,
					},
				},
			},
			wantErr: "run command on scale set rp-vmss-test instance 3 reported an error: first; second",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := runCommandResultError(vmScaleSetName, instanceID, tt.result)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
