package containerregistry

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
)

// ReplicationsClient is a minimal interface for azure ReplicationsClient
type ReplicationsClient interface {
	ReplicationsAddons
}

type replicationsClient struct {
	mgmtcontainerregistry.ReplicationsClient
}

var _ ReplicationsClient = &replicationsClient{}

// NewReplicationsClient creates a new ReplicationsClient
func NewReplicationsClient(subscriptionID string, authorizer autorest.Authorizer) ReplicationsClient {
	client := mgmtcontainerregistry.NewReplicationsClient(subscriptionID)
	client.Authorizer = authorizer

	return &replicationsClient{
		ReplicationsClient: client,
	}
}
