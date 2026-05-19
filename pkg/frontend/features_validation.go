package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armfeatures"
)

type FeaturesValidator interface {
	ValidateSubscriptionFeatures(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error
}

type featuresValidator struct{}

func (f featuresValidator) ValidateSubscriptionFeatures(ctx context.Context, azEnv *azureclient.AROEnvironment, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error {
	var fieldPath string
	if oc.Properties.MasterProfile.EncryptionAtHost == api.EncryptionAtHostEnabled {
		fieldPath = "properties.masterProfile.encryptionAtHost"
	} else if oc.Properties.WorkerProfiles[0].EncryptionAtHost == api.EncryptionAtHostEnabled {
		fieldPath = "properties.workerProfiles[0].encryptionAtHost"
	}

	if fieldPath != "" {
		fpCred, err := environment.FPNewClientCertificateCredential(tenantID, nil)
		if err != nil {
			return err
		}

		featuresClient, err := armfeatures.NewFeaturesClient(subscriptionID, fpCred, azEnv.ArmClientOptions())
		if err != nil {
			return err
		}

		return validateEncryptionAtHostFeature(ctx, featuresClient, subscriptionID, fieldPath)
	}
	return nil
}

func validateEncryptionAtHostFeature(
	ctx context.Context,
	featuresClient armfeatures.FeaturesClient,
	subscriptionID, fieldPath string,
) error {
	resp, err := featuresClient.Get(ctx, "Microsoft.Compute", "EncryptionAtHost", nil)
	if err != nil {
		return err
	}

	if resp.Properties == nil || resp.Properties.State == nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError,
			"",
			fmt.Sprintf(
				"Microsoft.Compute/EncryptionAtHost"+
					" feature has no state for"+
					" subscription %s.",
				subscriptionID))
	}

	if *resp.Properties.State != "Registered" {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			fieldPath,
			fmt.Sprintf(
				"Microsoft.Compute/EncryptionAtHost feature is not registered on subscription %s. "+
					"Please register the feature on your subscription before creating the cluster.",
				subscriptionID))
	}

	return nil
}
