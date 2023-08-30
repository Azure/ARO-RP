package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
)

type EncryptionAtHostValidator interface {
	ValidateEncryptionAtHost(ctx context.Context, subscriptionID string, oc *api.OpenShiftCluster) error
}

type encryptionAtHostValidator struct{}

func (e encryptionAtHostValidator) ValidateEncryptionAtHost(ctx context.Context, subscriptionID string, oc *api.OpenShiftCluster) error {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return err
	}

	subFeatureRegistrationsClient, err := armfeatures.NewSubscriptionFeatureRegistrationsClient(subscriptionID, credential, nil)
	if err != nil {
		return err
	}
	return validateEncryptionAtHost(ctx, subFeatureRegistrationsClient, oc)
}

func validateEncryptionAtHost(ctx context.Context, subFeatureRegistrationsClient features.SubscriptionFeatureRegistrationsClient, oc *api.OpenShiftCluster) error {
	encryptionSettings := make(map[api.EncryptionAtHost]bool)

	encryptionSettings[oc.Properties.MasterProfile.EncryptionAtHost] = true

	for _, wp := range oc.Properties.WorkerProfiles {
		encryptionSettings[wp.EncryptionAtHost] = true
	}

	for setting := range encryptionSettings {
		if setting == api.EncryptionAtHostEnabled {
			if err := IsRegisteredForEncryptionAtHostFeature(ctx, subFeatureRegistrationsClient); err != nil {
				return err
			}
		}
	}

	return nil
}

func IsRegisteredForEncryptionAtHostFeature(ctx context.Context, subFeatureRegistrationsClient features.SubscriptionFeatureRegistrationsClient) error {
	response, err := subFeatureRegistrationsClient.Get(ctx, "Microsoft.Compute", "EncryptionAtHost", nil)
	if err != nil {
		return err
	}
	if *response.Properties.State == armfeatures.SubscriptionFeatureRegistrationStateRegistered {
		return nil
	}
	return &api.CloudError{
		StatusCode: http.StatusBadRequest,
		CloudErrorBody: &api.CloudErrorBody{
			Code:    api.CloudErrorCodeInvalidParameter,
			Message: "Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
			Target:  "armfeatures.SubscriptionFeatureRegistrationProperties",
		},
	}
}
