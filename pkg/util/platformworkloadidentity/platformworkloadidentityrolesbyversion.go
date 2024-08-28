package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// PlatformWorkloadIdentityRolesByVersion is the interface that validates and obtains the version from an PlatformWorkloadIdentityRoleSetDocument.
type PlatformWorkloadIdentityRolesByVersion interface {
	GetPlatformWorkloadIdentityRolesByRoleName() map[string]api.PlatformWorkloadIdentityRole
}

// platformWorkloadIdentityRolesByVersionService is the default implementation of the PlatformWorkloadIdentityRolesByVersion interface.
type platformWorkloadIdentityRolesByVersionService struct {
	platformWorkloadIdentityRoles []api.PlatformWorkloadIdentityRole
}

// NewPlatformWorkloadIdentityRolesByVersion aims to populate platformWorkloadIdentityRoles for current OpenShift minor version and also for UpgradeableTo minor version if provided and is greater than the current version
func NewPlatformWorkloadIdentityRolesByVersion(ctx context.Context, oc *api.OpenShiftCluster, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets) (PlatformWorkloadIdentityRolesByVersion, error) {
	if !oc.UsesWorkloadIdentity() {
		return nil, nil
	}

	currentOpenShiftVersion, err := version.ParseVersion(oc.Properties.ClusterProfile.Version)
	if err != nil {
		return nil, err
	}
	currentMinorVersion := currentOpenShiftVersion.MinorVersion()
	requiredMinorVersions := map[string]bool{currentMinorVersion: false}
	platformWorkloadIdentityRoles := []api.PlatformWorkloadIdentityRole{}

	docs, err := dbPlatformWorkloadIdentityRoleSets.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	if oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo != nil {
		upgradeableVersion, err := version.ParseVersion(string(*oc.Properties.PlatformWorkloadIdentityProfile.UpgradeableTo))
		if err != nil {
			return nil, err
		}
		upgradeableMinorVersion := upgradeableVersion.MinorVersion()
		if currentMinorVersion != upgradeableMinorVersion && currentOpenShiftVersion.Lt(upgradeableVersion) {
			requiredMinorVersions[upgradeableMinorVersion] = false
		}
	}

	for _, doc := range docs.PlatformWorkloadIdentityRoleSetDocuments {
		for version := range requiredMinorVersions {
			if version == doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion {
				platformWorkloadIdentityRoles = append(platformWorkloadIdentityRoles, doc.PlatformWorkloadIdentityRoleSet.Properties.PlatformWorkloadIdentityRoles...)
				requiredMinorVersions[version] = true
			}
		}
	}

	for version, exists := range requiredMinorVersions {
		if !exists {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "No PlatformWorkloadIdentityRoleSet found for the requested or upgradeable OpenShift minor version '%s'. Please retry with different OpenShift version, and if the issue persists, raise an Azure support ticket", version)
		}
	}

	return &platformWorkloadIdentityRolesByVersionService{
		platformWorkloadIdentityRoles: platformWorkloadIdentityRoles,
	}, nil
}

func (service *platformWorkloadIdentityRolesByVersionService) GetPlatformWorkloadIdentityRolesByRoleName() map[string]api.PlatformWorkloadIdentityRole {
	platformWorkloadIdentityRolesByRoleName := map[string]api.PlatformWorkloadIdentityRole{}
	for _, role := range service.platformWorkloadIdentityRoles {
		platformWorkloadIdentityRolesByRoleName[role.OperatorName] = role
	}
	return platformWorkloadIdentityRolesByRoleName
}
