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

func (f *frontend) listSupportedVMSizes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	//r.URL.Path = filepath.Dir(r.URL.Path)
	b, err := f._listSupportedVMSizes(log, ctx, r)
	reply(log, w, nil, b, err)
}

func (f *frontend) _listSupportedVMSizes(log *logrus.Entry, ctx context.Context, r *http.Request) ([]byte, error) {
	var vmsizes map[api.VMSize]api.VMSizeStruct
	instanceType := r.URL.Query().Get("instanceType")
	switch instanceType {
	case "master":
		vmsizes = validate.SupportedMasterVmSizes
	case "worker":
		vmsizes = validate.SupportedWorkerVmSizes
	default:
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided instanceType '%s' is invalid. InstanceType can only be master or worker", instanceType)
	}
	b, err := json.MarshalIndent(vmsizes, "", "    ")
	if err != nil {
		return b, err
	}
	return b, nil
}
