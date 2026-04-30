package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
)

type openShiftVersionStaticValidator struct{}

// Validate validates an OpenShift cluster
func (sv openShiftVersionStaticValidator) Static(_new interface{}, _current *api.OpenShiftVersion) error {
	new := _new.(*OpenShiftVersion)

	var current *OpenShiftVersion
	if _current != nil {
		current = (&openShiftVersionConverter{}).ToExternal(_current).(*OpenShiftVersion)
	}

	err := sv.validate(new)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return sv.validateDelta(new, current)
}

func (sv openShiftVersionStaticValidator) validate(new *OpenShiftVersion) error {
	if new.Properties.Version == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.version", "Must be provided")
	}

	if new.Properties.InstallerPullspec == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.installerPullspec", "Must be provided")
	}

	if new.Properties.OpenShiftPullspec == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.openShiftPullspec", "Must be provided")
	}
	return nil
}

func (sv openShiftVersionStaticValidator) validateDelta(new, current *OpenShiftVersion) error {
	err := immutable.Validate("", new, current)
	if err != nil {
		err := err.(*immutable.ValidationError)
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, err.Target, err.Message)
	}
	return nil
}
