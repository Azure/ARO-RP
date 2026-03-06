package armcontainerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
)

type RegistriesClient interface {
	RegistriesClientAddons
}

type registriesClient struct {
	*armcontainerregistry.RegistriesClient
}

var _ RegistriesClient = &registriesClient{}

func NewRegistriesClient(subscriptionId string, credential azcore.TokenCredential, options *arm.ClientOptions) (RegistriesClient, error) {
	clientFactory, err := armcontainerregistry.NewClientFactory(subscriptionId, credential, options)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewRegistriesClient()

	return &registriesClient{
		RegistriesClient: client,
	}, nil
}
