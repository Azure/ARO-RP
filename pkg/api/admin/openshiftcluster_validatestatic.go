package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
)

type openShiftClusterStaticValidator struct{}

// Validate validates an OpenShift cluster
func (sv openShiftClusterStaticValidator) Static(_oc interface{}, _current *api.OpenShiftCluster, location, domain string, requireD2sV3Workers bool, resourceID string) error {
	if _current == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Admin API does not allow cluster creation.")
	}

	oc := _oc.(*OpenShiftCluster)
	return sv.validateDelta(oc, (&openShiftClusterConverter{}).ToExternal(_current).(*OpenShiftCluster))
}

func (sv openShiftClusterStaticValidator) validateDelta(oc, current *OpenShiftCluster) error {
	// Make a copy of the objects to modify without affecting originals
	ocCopy := *oc
	currentCopy := *current

	// Allow PreconfiguredNSG to change by resetting it before validation
	ocCopy.Properties.NetworkProfile.PreconfiguredNSG = currentCopy.Properties.NetworkProfile.PreconfiguredNSG

	// Run immutability validation, excluding PreconfiguredNSG
	err := immutable.Validate("", &ocCopy, &currentCopy)
	if err != nil {
		if validationErr, ok := err.(*immutable.ValidationError); ok {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, validationErr.Target, validationErr.Message)
		}
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "", err.Error())
	}

	return validateMaintenanceTask(oc.Properties.MaintenanceTask)
}

func validateMaintenanceTask(task MaintenanceTask) error {
	if !(task == "" ||
		task == MaintenanceTaskEverything ||
		task == MaintenanceTaskOperator ||
		task == MaintenanceTaskRenewCerts ||
		task == MaintenanceTaskPending ||
		task == MaintenanceTaskNone ||
		task == MaintenanceTaskSyncClusterObject ||
		task == MaintenanceTaskCustomerActionNeeded) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.maintenanceTask", "Invalid enum parameter.")
	}

	return nil
}
