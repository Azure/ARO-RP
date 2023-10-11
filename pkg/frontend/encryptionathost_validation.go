package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
)

type EncryptionAtHostValidator interface {
	ValidateEncryptionAtHost(ctx context.Context, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error
}

type encryptionAtHostValidator struct{}

func (e encryptionAtHostValidator) ValidateEncryptionAtHost(ctx context.Context, environment env.Interface, subscriptionID, tenantID string, oc *api.OpenShiftCluster) error {
	credential, err := environment.FPNewClientCertificateCredential(tenantID)
	if err != nil {
		return err
	}

	subFeatureRegistrationsClient, err := features.NewSubscriptionFeatureRegistrationsClient(subscriptionID, credential, nil)
	if err != nil {
		return err
	}
	return validateEncryptionAtHost(ctx, subFeatureRegistrationsClient, oc)
}

func validateEncryptionAtHost(ctx context.Context, subFeatureRegistrationsClient features.SubscriptionFeatureRegistrationsClient, oc *api.OpenShiftCluster) error {
	var hasEncryptionAtHostEnabled bool
	profilesToCheck := append([]api.WorkerProfile{{EncryptionAtHost: oc.Properties.MasterProfile.EncryptionAtHost}}, oc.Properties.WorkerProfiles...)
	for _, profile := range profilesToCheck {
		if profile.EncryptionAtHost == api.EncryptionAtHostEnabled {
			hasEncryptionAtHostEnabled = true
			break
		}
	}
	if hasEncryptionAtHostEnabled {
		err := IsRegisteredForEncryptionAtHostFeature(ctx, subFeatureRegistrationsClient)
		if err != nil {
			return err
		}
	}

	return nil
}

func IsRegisteredForEncryptionAtHostFeature(ctx context.Context, subFeatureRegistrationsClient features.SubscriptionFeatureRegistrationsClient) error {
	response, err := subFeatureRegistrationsClient.Get(ctx, "Microsoft.Compute", "EncryptionAtHost", nil)
	if err != nil {
		return err
	}
	if *response.Properties.State != armfeatures.SubscriptionFeatureRegistrationStateRegistered {
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeInvalidParameter,
				Message: "Microsoft.Compute/EncryptionAtHost feature is not enabled for this subscription. Register the feature using 'az feature register --namespace Microsoft.Compute --name EncryptionAtHost'",
				Target:  "armfeatures.SubscriptionFeatureRegistrationProperties",
			},
		}
	}
	return nil
}
