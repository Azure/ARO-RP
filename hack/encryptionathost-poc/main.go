package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"
)

func validateEncryptionAtHostFeature(ctx context.Context, subscriptionID string, featuresClient *armfeatures.Client) error {
	// Get the feature registration status
	resp, err := featuresClient.Get(ctx, "Microsoft.Compute", "EncryptionAtHost", nil)
	if err != nil {
		return fmt.Errorf("failed to get feature: %w", err)
	}

	if resp.Properties == nil || resp.Properties.State == nil {
		return fmt.Errorf("Microsoft.Compute/EncryptionAtHost feature has no state for subscription %s", subscriptionID)
	}

	if *resp.Properties.State != "Registered" {
		return fmt.Errorf("Microsoft.Compute/EncryptionAtHost is not registered for subscription %s (current state: %s)",
			subscriptionID, *resp.Properties.State)
	}

	return nil
}

func main() {
	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	registeredSubscriptionID := os.Getenv("SUBSCRIPTION_ID")
	notRegisteredSubscriptionID := os.Getenv("SUBSCRIPTION_ID_NOT_REGISTERED")

	// Create credential using track-2 SDK
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create credential: %v", err))
	}

	ctx := context.Background()

	// SUCCESS SCENARIO - feature IS registered
	fmt.Println("=== SCENARIO 1: Feature IS registered ===")
	featuresClient1, err := armfeatures.NewClient(registeredSubscriptionID, cred, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create features client: %v", err))
	}

	err = validateEncryptionAtHostFeature(ctx, registeredSubscriptionID, featuresClient1)
	if err != nil {
		fmt.Printf("FAIL - %v\n", err)
	} else {
		fmt.Println("SUCCESS - Feature is registered, cluster provisioning would proceed")
	}

	fmt.Println("")

	// FAILURE SCENARIO - feature is NOT registered
	fmt.Println("=== SCENARIO 2: Feature is NOT registered ===")
	featuresClient2, err := armfeatures.NewClient(notRegisteredSubscriptionID, cred, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create features client: %v", err))
	}

	err = validateEncryptionAtHostFeature(ctx, notRegisteredSubscriptionID, featuresClient2)
	if err != nil {
		fmt.Printf("SUCCESS - Correctly caught unregistered feature: %v\n", err)
	} else {
		fmt.Println("FAIL - Should have returned error but did not")
	}
}
