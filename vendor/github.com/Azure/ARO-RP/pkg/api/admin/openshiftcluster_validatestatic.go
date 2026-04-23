package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"slices"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
)

type openShiftClusterStaticValidator struct{}

// Validate validates an OpenShift cluster
func (sv openShiftClusterStaticValidator) Static(_oc interface{}, _current *api.OpenShiftCluster, location, domain string, requireD2sWorkers bool, installArchitectureVersion api.ArchitectureVersion, resourceID string) error {
	if _current == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Admin API does not allow cluster creation.")
	}

	oc := _oc.(*OpenShiftCluster)
	return sv.validateDelta(oc, (&openShiftClusterConverter{}).ToExternal(_current).(*OpenShiftCluster))
}

func (sv openShiftClusterStaticValidator) validateDelta(oc, current *OpenShiftCluster) error {
	oc.Properties.NetworkProfile.PreconfiguredNSG = current.Properties.NetworkProfile.PreconfiguredNSG
	err := immutable.Validate("", oc, current)
	if err != nil {
		if validationErr, ok := err.(*immutable.ValidationError); ok {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, validationErr.Target, validationErr.Message)
		}
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "", err.Error())
	}

	return validateMaintenanceTask(oc.Properties.MaintenanceTask)
}

func validateMaintenanceTask(task MaintenanceTask) error {
	if task == "" || slices.Contains(validMaintenanceTasks, task) {
		return nil
	}

	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.maintenanceTask", "Invalid enum parameter.")
}
