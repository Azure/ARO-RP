package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

func (dv *dynamic) ValidateEncryptionAtHost(ctx context.Context, oc *api.OpenShiftCluster) error {
	dv.log.Print("ValidateEncryptionAtHost")

	if oc.Properties.MasterProfile.EncryptionAtHost == api.EncryptionAtHostEnabled {
		err := dv.validateEncryptionAtHostSupport(oc.Properties.MasterProfile.VMSize, "properties.masterProfile.encryptionAtHost")
		if err != nil {
			return err
		}
	}

	workerProfiles, propertyName := api.GetEnrichedWorkerProfiles(oc.Properties)
	for i, wp := range workerProfiles {
		if wp.EncryptionAtHost == api.EncryptionAtHostEnabled {
			err := dv.validateEncryptionAtHostSupport(wp.VMSize, fmt.Sprintf("properties.%s[%d].encryptionAtHost", propertyName, i))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dv *dynamic) validateEncryptionAtHostSupport(VMSize api.VMSize, path string) error {
	sku, err := dv.env.VMSku(string(VMSize))
	if err != nil {
		return err
	}

	if !computeskus.HasCapability(sku, "EncryptionAtHostSupported") {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, fmt.Sprintf("VM SKU '%s' does not support encryption at host.", VMSize))
	}

	return nil
}
