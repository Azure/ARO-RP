package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/vms"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) supportedvmsizes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vmRole := vms.VMRole(r.URL.Query().Get("vmRole"))
	b, err := f.supportedVMSizesForRole(vmRole)
	reply(log, w, nil, b, err)
}

func (f *frontend) supportedVMSizesForRole(vmRole vms.VMRole) ([]byte, error) {
	if vmRole != vms.VMRoleMaster && vmRole != vms.VMRoleWorker {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", fmt.Sprintf("The provided vmRole '%s' is invalid. vmRole can only be master or worker", vmRole))
	}
	vmsizes := vms.SupportedVMSizesByRole[vmRole]
	b, err := json.MarshalIndent(vmsizes, "", "    ")
	if err != nil {
		return b, err
	}
	return b, nil
}
