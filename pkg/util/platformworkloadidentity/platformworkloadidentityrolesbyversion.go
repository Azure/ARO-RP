package platformworkloadidentity

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
)

// PlatformWorkloadIdentityRolesByVersion is the interface that validates and obtains the version from an PlatformWorkloadIdentityRoleSetDocument.
type PlatformWorkloadIdentityRolesByVersion interface {
	GetPlatformWorkloadIdentityRoles() []api.PlatformWorkloadIdentityRole
}

// platformWorkloadIdentityRolesByVersionService is the default implementation of the PlatformWorkloadIdentityRolesByVersion interface.
type platformWorkloadIdentityRolesByVersionService struct {
	platformWorkloadIdentityRoles []api.PlatformWorkloadIdentityRole
}

func NewPlatformWorkloadIdentityRolesByVersion(ctx context.Context, oc *api.OpenShiftCluster, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets) (PlatformWorkloadIdentityRolesByVersion, error) {
	if oc.Properties.PlatformWorkloadIdentityProfile == nil || oc.Properties.ServicePrincipalProfile != nil {
		return nil, nil
	}

	requestedInstallVersion := oc.Properties.ClusterProfile.Version

	docs, err := dbPlatformWorkloadIdentityRoleSets.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, doc := range docs.PlatformWorkloadIdentityRoleSetDocuments {
		if requestedInstallVersion == doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion {
			return &platformWorkloadIdentityRolesByVersionService{
				platformWorkloadIdentityRoles: doc.PlatformWorkloadIdentityRoleSet.Properties.PlatformWorkloadIdentityRoles,
			}, nil
		}
	}

	return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", "No PlatformWorkloadIdentityRoleSet found for the requested OpenShift version '%s'.", requestedInstallVersion)
}

func (service *platformWorkloadIdentityRolesByVersionService) GetPlatformWorkloadIdentityRoles() []api.PlatformWorkloadIdentityRole {
	return service.platformWorkloadIdentityRoles
}
