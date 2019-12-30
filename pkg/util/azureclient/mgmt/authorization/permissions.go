package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../../vendor/github.com/golang/mock/mockgen -destination=../../../../util/mocks/azureclient/mgmt/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/$GOPACKAGE PermissionsClient,RoleAssignmentsClient
//go:generate go run ../../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../../util/mocks/azureclient/mgmt/$GOPACKAGE/$GOPACKAGE.go

import (
	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest"
)

// PermissionsClient is a minimal interface for azure PermissionsClient
type PermissionsClient interface {
	PermissionsClientAddons
}

type permissionsClient struct {
	authorization.PermissionsClient
}

var _ PermissionsClient = &permissionsClient{}

// NewPermissionsClient creates a new PermissionsClient
func NewPermissionsClient(subscriptionID string, authorizer autorest.Authorizer) PermissionsClient {
	client := authorization.NewPermissionsClient(subscriptionID)
	client.Authorizer = authorizer

	return &permissionsClient{
		PermissionsClient: client,
	}
}
