package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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

func (f *frontend) listSupportedVMSizes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	b, err := f._listSupportedVMSizes(log, ctx, r)
	reply(log, w, nil, b, err)
}

func (f *frontend) _listSupportedVMSizes(log *logrus.Entry, ctx context.Context, r *http.Request) ([]byte, error) {
	vmRole := r.URL.Query().Get("vmRole")
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
