package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

type featureResult struct {
	Properties *featureResultProperties `json:"properties"`
}

type featureResultProperties struct {
	State string `json:"state"`
}

func validateEncryptionAtHostFeature(subscriptionID string, authorizer autorest.Authorizer) error {
	client := autorest.NewClientWithUserAgent("")
	client.Authorizer = authorizer

	url := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/providers/Microsoft.Features/providers/Microsoft.Compute/features/EncryptionAtHost",
		subscriptionID)

	req, err := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(url),
		autorest.WithQueryParameters(map[string]interface{}{
			"api-version": "2021-07-01",
		}),
	).Prepare((&http.Request{}).WithContext(context.Background()))
	if err != nil {
		return fmt.Errorf("failed to prepare request: %v", err)
	}

	resp, err := autorest.SendWithSender(client, req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	var result featureResult
	err = autorest.Respond(resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if result.Properties == nil ||
		result.Properties.State != "Registered" {
		return fmt.Errorf("Microsoft.Compute/EncryptionAtHost is not registered for subscription %s", subscriptionID)
	}

	return nil
}

func main() {
	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	registeredSubscriptionID := os.Getenv("SUBSCRIPTION_ID")
	notRegisteredSubscriptionID := os.Getenv("SUBSCRIPTION_ID_NOT_REGISTERED")

	oauthConfig, err := adal.NewOAuthConfig(
		azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		panic(err)
	}

	token, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		clientID,
		clientSecret,
		azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		panic(err)
	}

	authorizer := autorest.NewBearerAuthorizer(token)

	// SUCCESS SCENARIO - feature IS registered
	fmt.Println("=== SCENARIO 1: Feature IS registered ===")
	err = validateEncryptionAtHostFeature(registeredSubscriptionID, authorizer)
	if err != nil {
		fmt.Printf("FAIL - %v\n", err)
	} else {
		fmt.Println("SUCCESS - Feature is registered, cluster provisioning would proceed")
	}

	fmt.Println("")

	// FAILURE SCENARIO - feature is NOT registered
	fmt.Println("=== SCENARIO 2: Feature is NOT registered ===")
	err = validateEncryptionAtHostFeature(notRegisteredSubscriptionID, authorizer)
	if err != nil {
		fmt.Printf("SUCCESS - Correctly caught unregistered feature: %v\n", err)
	} else {
		fmt.Println("FAIL - Should have returned error but did not")
	}
}
