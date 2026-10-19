package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// CapacityReservationGroupsClient wraps the Azure SDK CapacityReservationGroupsClient,
// exposing only the methods needed for CRG lifecycle management during VM resize.
type CapacityReservationGroupsClient interface {
	CapacityReservationGroupsClientAddons
}

type capacityReservationGroupsClient struct {
	*armcompute.CapacityReservationGroupsClient
}

var _ CapacityReservationGroupsClient = &capacityReservationGroupsClient{}

// NewDefaultCapacityReservationGroupsClient creates a CapacityReservationGroupsClient using the ARO environment's cloud configuration.
func NewDefaultCapacityReservationGroupsClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (CapacityReservationGroupsClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}
	return NewCapacityReservationGroupsClient(subscriptionID, credential, options)
}

// NewCapacityReservationGroupsClient creates a CapacityReservationGroupsClient with the supplied ARM client options.
func NewCapacityReservationGroupsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (CapacityReservationGroupsClient, error) {
	clientFactory, err := armcompute.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &capacityReservationGroupsClient{
		CapacityReservationGroupsClient: clientFactory.NewCapacityReservationGroupsClient(),
	}, nil
}
