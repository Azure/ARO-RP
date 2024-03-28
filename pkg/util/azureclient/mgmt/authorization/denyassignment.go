package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	moduleName    = "github.com/Azure/ARO-RP/pkg/cluster/delete.go"
	moduleVersion = "0.0.1"
)

// DenyAssignmentClient is a minimal interface for azure DenyAssignmentClient
type DenyAssignmentClient interface {
	DenyAssignmentClientAddons
}

type DenyAssignmentsARMClient struct {
	internal       *arm.Client
	subscriptionID string
}

type denyAssignmentClient struct {
	mgmtauthorization.DenyAssignmentsClient
}

var _ DenyAssignmentClient = &denyAssignmentClient{}

// NewDenyAssignmentsClient creates a new DenyAssignmentsClient
func NewDenyAssignmentsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) DenyAssignmentClient {
	client := mgmtauthorization.NewDenyAssignmentsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &denyAssignmentClient{
		DenyAssignmentsClient: client,
	}
}

// New deny assignment client similar to other clients in https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/resourcemanager/authorization/armauthorization/denyassignments_client.go
func NewDenyAssignmentsARMClient(subscriptionID string, credential *azidentity.ClientCertificateCredential, options *arm.ClientOptions) (*DenyAssignmentsARMClient, error) {
	cl, err := arm.NewClient(moduleName, moduleVersion, credential, options)
	if err != nil {
		return nil, err
	}
	client := &DenyAssignmentsARMClient{
		subscriptionID: subscriptionID,
		internal:       cl,
	}
	return client, nil
}
