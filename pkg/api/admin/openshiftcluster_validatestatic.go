package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/immutable"
)

type openShiftClusterStaticValidator struct{}

// Validate validates an OpenShift cluster
func (sv *openShiftClusterStaticValidator) Static(_oc interface{}, _current *api.OpenShiftCluster) error {
	if _current == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Admin API does not allow cluster creation.")
	}

	oc, ok := _oc.(*OpenShiftCluster)
	if !ok {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
	}
	return sv.validateDelta(oc, (&openShiftClusterConverter{}).ToExternal(_current).(*OpenShiftCluster))
}

func (sv *openShiftClusterStaticValidator) validateDelta(oc, current *OpenShiftCluster) error {
	err := immutable.Validate("", oc, current)
	if err != nil {
		err, ok := err.(*immutable.ValidationError)
		if !ok {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		}
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, err.Target, err.Message)
	}

	return nil
}
