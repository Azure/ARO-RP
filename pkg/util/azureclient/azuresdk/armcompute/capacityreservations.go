package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// CapacityReservationsClient wraps the Azure SDK CapacityReservationsClient,
// exposing only the methods needed for capacity reservation lifecycle management during VM resize.
type CapacityReservationsClient interface {
	CapacityReservationsClientAddons
}

type capacityReservationsClient struct {
	*armcompute.CapacityReservationsClient
}

var _ CapacityReservationsClient = &capacityReservationsClient{}

// NewDefaultCapacityReservationsClient creates a CapacityReservationsClient using the ARO environment's cloud configuration.
func NewDefaultCapacityReservationsClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (CapacityReservationsClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	return NewCapacityReservationsClient(subscriptionID, credential, options)
}

// NewCapacityReservationsClient creates a CapacityReservationsClient with the supplied ARM client options.
func NewCapacityReservationsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (CapacityReservationsClient, error) {
	clientFactory, err := armcompute.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &capacityReservationsClient{
		CapacityReservationsClient: clientFactory.NewCapacityReservationsClient(),
	}, nil
}
