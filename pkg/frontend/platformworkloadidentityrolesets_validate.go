package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
)

// validatePlatformWorkloadIdentities validates that customer provided platform workload identities are expected
func (f *frontend) validatePlatformWorkloadIdentities(oc *api.OpenShiftCluster) error {
	roleSets := f.getAvailablePlatformWorkloadIdentityRoleSets()

	platformWorkloadIdentityRolesByVersionService := platformworkloadidentity.NewPlatformWorkloadIdentityRolesByVersionService()
	err := platformWorkloadIdentityRolesByVersionService.PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets(oc, roleSets)
	if err != nil {
		return err
	}
	matches := platformWorkloadIdentityRolesByVersionService.MatchesPlatformWorkloadIdentityRoles(oc)

	if !matches {
		return platformworkloadidentity.GetPlatformWorkloadIdentityMismatchError(oc, platformWorkloadIdentityRolesByVersionService.GetPlatformWorkloadIdentityRolesByRoleName())
	}

	return nil
}
