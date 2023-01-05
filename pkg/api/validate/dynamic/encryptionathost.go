package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

type EncryptionAtHostValidator interface {
	Validate(ctx context.Context, oc *api.OpenShiftCluster) error
}

type defaultEncrytpionValidator struct {
	env env.Interface
	log *logrus.Entry
}

func NewEncryptionAtHostValidator(env env.Interface, log *logrus.Entry) defaultEncrytpionValidator {
	return defaultEncrytpionValidator{
		env: env,
		log: log,
	}
}

func (dv defaultEncrytpionValidator) Validate(ctx context.Context, oc *api.OpenShiftCluster) error {
	dv.log.Print("ValidateEncryptionAtHost")

	if oc.Properties.MasterProfile.EncryptionAtHost == api.EncryptionAtHostEnabled {
		err := dv.validateEncryptionAtHostSupport(oc.Properties.MasterProfile.VMSize, "properties.masterProfile.encryptionAtHost")
		if err != nil {
			return err
		}
	}

	for i, wp := range oc.Properties.WorkerProfiles {
		if wp.EncryptionAtHost == api.EncryptionAtHostEnabled {
			err := dv.validateEncryptionAtHostSupport(wp.VMSize, fmt.Sprintf("properties.workerProfiles[%d].encryptionAtHost", i))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dv defaultEncrytpionValidator) validateEncryptionAtHostSupport(VMSize api.VMSize, path string) error {
	sku, err := dv.env.VMSku(string(VMSize))
	if err != nil {
		return err
	}

	if !computeskus.HasCapability(sku, "EncryptionAtHostSupported") {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path, "VM SKU '%s' does not support encryption at host.", VMSize)
	}

	return nil
}
