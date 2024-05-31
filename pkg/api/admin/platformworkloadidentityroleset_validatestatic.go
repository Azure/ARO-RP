package admin

import (
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type platformWorkloadIdentityRoleSetStaticValidator struct{}

func (sv platformWorkloadIdentityRoleSetStaticValidator) Static(_new interface{}, _current *api.PlatformWorkloadIdentityRoleSet) error {
	new := _new.(*PlatformWorkloadIdentityRoleSet)

	var current *PlatformWorkloadIdentityRoleSet
	if _current != nil {
		current = (&platformWorkloadIdentityRoleSetConverter{}).ToExternal(_current).(*PlatformWorkloadIdentityRoleSet)
	}

	err := sv.validate(new, current == nil)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return sv.validateDelta(new, current)
}

func (sv platformWorkloadIdentityRoleSetStaticValidator) validate(new *PlatformWorkloadIdentityRoleSet, isCreate bool) error {
	if new.Properties.OpenShiftVersion == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.openShiftVersion", "Must be provided")
	}

	if new.Properties.PlatformWorkloadIdentityRoles == nil || len(new.Properties.PlatformWorkloadIdentityRoles) == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.platformWorkloadIdentityRoles", "Must be provided and must be non-empty")
	}

	for i, r := range new.Properties.PlatformWorkloadIdentityRoles {
		if r.OperatorName == "" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("properties.platformWorkloadIdentityRoles[%d].operatorName", i), "Must be provided")
		}

		if r.RoleDefinitionName == "" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("properties.platformWorkloadIdentityRoles[%d].roleDefinitionName", i), "Must be provided")
		}

		if r.RoleDefinitionID == "" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("properties.platformWorkloadIdentityRoles[%d].roleDefinitionId", i), "Must be provided")
		}

		if r.ServiceAccounts == nil || len(r.ServiceAccounts) == 0 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("properties.platformWorkloadIdentityRoles[%d].serviceAccounts", i), "Must be provided and must be non-empty")
		}
	}

	return nil
}

func (sv platformWorkloadIdentityRoleSetStaticValidator) validateDelta(new, current *PlatformWorkloadIdentityRoleSet) error {
	err := immutable.Validate("", new, current)
	if err != nil {
		err := err.(*immutable.ValidationError)
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, err.Target, err.Message)
	}
	return nil
}
