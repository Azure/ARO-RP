package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

var validVMRoles = map[string]map[api.VMSize]api.VMSizeStruct{
	"master": validate.SupportedMasterVmSizes,
	"worker": validate.SupportedWorkerVmSizes,
}

func (f *frontend) supportedvmsizes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vmRole := r.URL.Query().Get("vmRole")
	b, err := f.supportedVMSizesForRole(vmRole)
	reply(log, w, nil, b, err)
}

func (f *frontend) supportedVMSizesForRole(vmRole string) ([]byte, error) {
	vmsizes, exists := validVMRoles[vmRole]
	if !exists {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided vmRole '%s' is invalid. vmRole can only be master or worker", vmRole)
	}
	b, err := json.MarshalIndent(vmsizes, "", "    ")
	if err != nil {
		return b, err
	}
	return b, nil
}
