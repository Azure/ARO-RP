package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
)

// validatePlatformWorkloadIdentities validates that customer provided platform workload identities are expected
func (f *frontend) validatePlatformWorkloadIdentities(oc *api.OpenShiftCluster) error {
	roleSets := make([]*api.PlatformWorkloadIdentityRoleSet, 0)

	f.platformWorkloadIdentityRoleSetsMu.RLock()
	for _, pwirs := range f.availablePlatformWorkloadIdentityRoleSets {
		roleSets = append(roleSets, pwirs)
	}

	platformWorkloadIdentityRolesByVersionService := platformworkloadidentity.NewPlatformWorkloadIdentityRolesByVersionService()
	platformWorkloadIdentityRolesByVersionService.PopulatePlatformWorkloadIdentityRolesByVersionUsingRoleSets(oc, roleSets)
	matches := platformWorkloadIdentityRolesByVersionService.MatchesPlatformWorkloadIdentityRoles(oc)

	f.platformWorkloadIdentityRoleSetsMu.RUnlock()

	if !matches {
		return platformworkloadidentity.GetPlatformWorkloadIdentityMismatchError(oc, platformWorkloadIdentityRolesByVersionService.GetPlatformWorkloadIdentityRolesByRoleName())
	}

	return nil
}
