package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// CapacityReservationGroupsClient is a minimal interface for armcompute CapacityReservationGroupsClient
type CapacityReservationGroupsClient interface {
	CapacityReservationGroupsClientAddons
}

type capacityReservationGroupsClient struct {
	*armcompute.CapacityReservationGroupsClient
}

var _ CapacityReservationGroupsClient = &capacityReservationGroupsClient{}

// NewDefaultCapacityReservationGroupsClient creates a new CapacityReservationGroupsClient with default options
func NewDefaultCapacityReservationGroupsClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (CapacityReservationGroupsClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	return NewCapacityReservationGroupsClient(subscriptionID, credential, options)
}

// NewCapacityReservationGroupsClient creates a new CapacityReservationGroupsClient
func NewCapacityReservationGroupsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (CapacityReservationGroupsClient, error) {
	clientFactory, err := armcompute.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &capacityReservationGroupsClient{
		CapacityReservationGroupsClient: clientFactory.NewCapacityReservationGroupsClient(),
	}, nil
}
