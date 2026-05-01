package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type featureResult struct {
	Properties *featureResultProperties `json:"properties"`
}

type featureResultProperties struct {
	State string `json:"state"`
}

func validateEncryptionAtHostFeature(
	ctx context.Context,
	azEnv *azureclient.AROEnvironment,
	environment env.Interface,
	subscriptionID, tenantID string,
) error {
	fpAuthorizer, err := environment.FPAuthorizer(
		tenantID, nil,
		environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}

	client := autorest.NewClientWithUserAgent("")
	client.Authorizer = fpAuthorizer

	url := fmt.Sprintf(
		"%ssubscriptions/%s/providers/Microsoft.Features/providers/Microsoft.Compute/features/EncryptionAtHost",
		azEnv.ResourceManagerEndpoint,
		subscriptionID)

	req, err := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(url),
		autorest.WithQueryParameters(map[string]interface{}{
			"api-version": "2021-07-01",
		}),
	).Prepare((&http.Request{}).WithContext(ctx))
	if err != nil {
		return err
	}

	resp, err := autorest.SendWithSender(client, req)
	if err != nil {
		return err
	}

	var result featureResult
	err = autorest.Respond(resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	if err != nil {
		return err
	}

	if result.Properties == nil ||
		result.Properties.State != "Registered" {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"encryptionAtHost",
			fmt.Sprintf(
				"Microsoft.Compute/EncryptionAtHost"+
					" is not registered for"+
					" subscription %s.",
				subscriptionID))
	}

	return nil
}
