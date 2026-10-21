package armredhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	armredhatopenshift "github.com/Azure/ARO-RP/pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift"
)

type OperationsClient interface {
	OperationsAddons
}

type operationsClient struct {
	*armredhatopenshift.OperationsClient
}

var _ OperationsClient = &operationsClient{}

func NewOperationsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (OperationsClient, error) {
	_ = subscriptionID
	options = configureDevMode(options)

	client, err := armredhatopenshift.NewOperationsClient(credential, options)
	if err != nil {
		return nil, err
	}

	return &operationsClient{
		OperationsClient: client,
	}, nil
}
