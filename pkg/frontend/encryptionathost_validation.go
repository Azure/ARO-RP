package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func validateEncryptionAtHostFeature(
	ctx context.Context,
	azEnv *azureclient.AROEnvironment,
	environment env.Interface,
	subscriptionID, tenantID string,
) error {
	fpCred, err := environment.FPNewClientCertificateCredential(tenantID, nil)
	if err != nil {
		return err
	}

	featuresClient, err := armfeatures.NewClient(subscriptionID, fpCred, azEnv.ArmClientOptions())
	if err != nil {
		return err
	}

	resp, err := featuresClient.Get(ctx, "Microsoft.Compute", "EncryptionAtHost", nil)
	if err != nil {
		return err
	}

	if resp.Properties == nil || resp.Properties.State == nil {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"encryptionAtHost",
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
			"encryptionAtHost",
			fmt.Sprintf(
				"Microsoft.Compute/EncryptionAtHost feature is not registered on subscription %s. "+
					"Please register the feature on your subscription before creating the cluster.",
				subscriptionID))
	}

	return nil
}
