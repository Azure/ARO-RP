package armpolicy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type AssignmentsClient interface {
	Create(ctx context.Context, scope string, policyAssignmentName string, parameters armpolicy.Assignment, options *armpolicy.AssignmentsClientCreateOptions) (armpolicy.AssignmentsClientCreateResponse, error)
	Delete(ctx context.Context, scope string, policyAssignmentName string, options *armpolicy.AssignmentsClientDeleteOptions) (armpolicy.AssignmentsClientDeleteResponse, error)
}

type assignmentsClient struct {
	*armpolicy.AssignmentsClient
}

func NewAssignmentsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (AssignmentsClient, error) {
	clientFactory, err := armpolicy.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &assignmentsClient{clientFactory.NewAssignmentsClient()}, nil
}
