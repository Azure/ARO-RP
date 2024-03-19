package failure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"regexp"
	"testing"
)

func TestInstallFailingReasonRegexes(t *testing.T) {
	for _, tt := range []struct {
		name       string
		installLog string
		want       InstallFailingReason
	}{
		{
			name: "InvalidTemplateDeployment - no known errors",
			installLog: `
level=info msg=running in local development mode
level=info msg=creating development InstanceMetadata
level=info msg=InstanceMetadata: running on AzurePublicCloud
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func1]
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func2]
level=info msg=resolving graph
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func3]
level=info msg=checking if graph exists
level=info msg=save graph
Generates the Ignition Config asset

level=info msg=running in local development mode
level=info msg=creating development InstanceMetadata
level=info msg=InstanceMetadata: running on AzurePublicCloud
level=info msg=running step [AuthorizationRetryingAction github.com/openshift/ARO-Installer/pkg/installer.(*manager).deployResourceTemplate-fm]
level=info msg=load persisted graph
level=info msg=deploying resources template
level=error msg=step [AuthorizationRetryingAction github.com/openshift/ARO-Installer/pkg/installer.(*manager).deployResourceTemplate-fm] encountered error: 400: DeploymentFailed: : Deployment failed. Details: : : {"code":"InvalidTemplateDeployment","message":"The template deployment failed with multiple errors. Please see details for more information.","details":[]}
level=error msg=400: DeploymentFailed: : Deployment failed. Details: : : {"code":"InvalidTemplateDeployment","message":"The template deployment failed with multiple errors. Please see details for more information.","details":[]}`,
			want: AzureInvalidTemplateDeployment,
		},
		{
			name: "InvalidTemplateDeployment - RequestDisallowedByPolicy",
			installLog: `
level=info msg=running in local development mode
level=info msg=creating development InstanceMetadata
level=info msg=InstanceMetadata: running on AzurePublicCloud
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func1]
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func2]
level=info msg=resolving graph
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func3]
level=info msg=checking if graph exists
level=info msg=save graph
Generates the Ignition Config asset

level=info msg=running in local development mode
level=info msg=creating development InstanceMetadata
level=info msg=InstanceMetadata: running on AzurePublicCloud
level=info msg=running step [AuthorizationRetryingAction github.com/openshift/ARO-Installer/pkg/installer.(*manager).deployResourceTemplate-fm]
level=info msg=load persisted graph
level=info msg=deploying resources template
level=error msg=step [AuthorizationRetryingAction github.com/openshift/ARO-Installer/pkg/installer.(*manager).deployResourceTemplate-fm] encountered error: 400: DeploymentFailed: : Deployment failed. Details: : : {"code":"InvalidTemplateDeployment","message":"The template deployment failed with multiple errors. Please see details for more information.","details":[{"additionalInfo":[],"code":"RequestDisallowedByPolicy","message":"Resource 'test-bootstrap' was disallowed by policy. Policy identifiers: ''.","target":"test-bootstrap"}]}
level=error msg=400: DeploymentFailed: : Deployment failed. Details: : : {"code":"InvalidTemplateDeployment","message":"The template deployment failed with multiple errors. Please see details for more information.","details":[{"additionalInfo":[],"code":"RequestDisallowedByPolicy","message":"Resource 'test-bootstrap' was disallowed by policy. Policy identifiers: ''.","target":"test-bootstrap"}]}`,
			want: AzureRequestDisallowedByPolicy,
		},
		{
			name: "ResourceDeploymentFailure - ZonalAllocationFailed",
			installLog: `
level=info msg=running in local development mode
level=info msg=creating development InstanceMetadata
level=info msg=InstanceMetadata: running on AzurePublicCloud
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func1]
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func2]
level=info msg=resolving graph
level=info msg=running step [Action github.com/openshift/ARO-Installer/pkg/installer.(*manager).Manifests.func3]
level=info msg=checking if graph exists
level=info msg=save graph
Generates the Ignition Config asset

level=info msg=running in local development mode
level=info msg=creating development InstanceMetadata
level=info msg=InstanceMetadata: running on AzurePublicCloud
level=info msg=running step [AuthorizationRetryingAction github.com/openshift/ARO-Installer/pkg/installer.(*manager).deployResourceTemplate-fm]
level=info msg=load persisted graph
level=info msg=deploying resources template
level=error msg=step [AuthorizationRetryingAction github.com/openshift/ARO-Installer/pkg/installer.(*manager).deployResourceTemplate-fm] encountered error: 400: DeploymentFailed: : Deployment failed. Details: : : {"code":"DeploymentFailed","message":"At least one resource deployment operation failed. Please list deployment operations for details. Please see https://aka.ms/arm-deployment-operations for usage details.","details":[{"additionalInfo":[],"code":"ZonalAllocationFailed","message":"Allocation failed. We do not have sufficient capacity for the requested VM size in this zone.","target":"null"}]}
level=error msg=400: DeploymentFailed: : Deployment failed. Details: : : {"code":"DeploymentFailed","message":"At least one resource deployment operation failed. Please list deployment operations for details. Please see https://aka.ms/arm-deployment-operations for usage details.","details":[{"additionalInfo":[],"code":"ZonalAllocationFailed","message":"Allocation failed. We do not have sufficient capacity for the requested VM size in this zone.","target":"null"}]}`,
			want: AzureZonalAllocationFailed,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// This test uses a "mock" version of Hive's real implementation for matching install logs against regex patterns.
			// https://github.com/bennerv/hive/blob/fec14dcf0-plus-base-image-update/pkg/controller/clusterprovision/installlogmonitor.go#L83
			// The purpose of this test is to test the regular expressions themselves, not the implementation.
			got := mockHiveIdentifyReason(tt.installLog)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func mockHiveIdentifyReason(installLog string) InstallFailingReason {
	for _, reason := range Reasons {
		for _, regex := range reason.SearchRegexes {
			if regex.MatchString(installLog) {
				return reason
			}
		}
	}

	return InstallFailingReason{
		Name:          "UnknownError",
		Reason:        "UnknownError",
		Message:       installLog,
		SearchRegexes: []*regexp.Regexp{},
	}
}
