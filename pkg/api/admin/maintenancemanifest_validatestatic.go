package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
)

type maintenanceManifestStaticValidator struct{}

// Validate validates a MaintenanceManifest
func (sv maintenanceManifestStaticValidator) Static(_new interface{}, _current *api.MaintenanceManifestDocument) error {
	new := _new.(*MaintenanceManifest)

	var current *MaintenanceManifest
	if _current != nil {
		current = (&maintenanceManifestConverter{}).ToExternal(_current, false).(*MaintenanceManifest)
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

func (sv maintenanceManifestStaticValidator) validate(new *MaintenanceManifest) error {
	if new.MaintenanceTaskID == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "maintenanceTaskID", "Must be provided")
	}

	if new.RunAfter == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "runAfter", "Must be provided")
	}

	if new.RunBefore == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "runBefore", "Must be provided")
	}

	return nil
}

func (sv maintenanceManifestStaticValidator) validateDelta(new, current *MaintenanceManifest) error {
	err := immutable.Validate("", new, current)
	if err != nil {
		err := err.(*immutable.ValidationError)
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, err.Target, err.Message)
	}
	return nil
}
