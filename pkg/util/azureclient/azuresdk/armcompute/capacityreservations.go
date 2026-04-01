package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// CapacityReservationsClient is a minimal interface for armcompute CapacityReservationsClient
type CapacityReservationsClient interface {
	CapacityReservationsClientAddons
}

type capacityReservationsClient struct {
	*armcompute.CapacityReservationsClient
}

var _ CapacityReservationsClient = &capacityReservationsClient{}

// NewDefaultCapacityReservationsClient creates a new CapacityReservationsClient with default options
func NewDefaultCapacityReservationsClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (CapacityReservationsClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	return NewCapacityReservationsClient(subscriptionID, credential, options)
}

// NewCapacityReservationsClient creates a new CapacityReservationsClient
func NewCapacityReservationsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (CapacityReservationsClient, error) {
	clientFactory, err := armcompute.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &capacityReservationsClient{
		CapacityReservationsClient: clientFactory.NewCapacityReservationsClient(),
	}, nil
}
