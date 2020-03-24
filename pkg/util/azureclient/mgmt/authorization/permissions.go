package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest"
)

// PermissionsClient is a minimal interface for azure PermissionsClient
type PermissionsClient interface {
	PermissionsClientAddons
}

type permissionsClient struct {
	mgmtauthorization.PermissionsClient
}

var _ PermissionsClient = &permissionsClient{}

// NewPermissionsClient creates a new PermissionsClient
func NewPermissionsClient(subscriptionID string, authorizer autorest.Authorizer) PermissionsClient {
	client := mgmtauthorization.NewPermissionsClient(subscriptionID)
	client.Authorizer = authorizer

	return &permissionsClient{
		PermissionsClient: client,
	}
}
