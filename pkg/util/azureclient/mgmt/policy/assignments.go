package policy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtpolicy "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-09-01/policy"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// AssignmentsClient is a minimal interface for azure AssignmentsClient
type AssignmentsClient interface {
	Create(ctx context.Context, scope string, policyAssignmentName string, parameters mgmtpolicy.Assignment) (result mgmtpolicy.Assignment, err error)
	Delete(ctx context.Context, scope string, policyAssignmentName string) (result mgmtpolicy.Assignment, err error)
	Get(ctx context.Context, scope string, policyAssignmentName string) (result mgmtpolicy.Assignment, err error)
}

type assignmentsClient struct {
	mgmtpolicy.AssignmentsClient
}

var _ AssignmentsClient = &assignmentsClient{}

func NewAssignmentsClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) AssignmentsClient {
	client := mgmtpolicy.NewAssignmentsClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer

	return &assignmentsClient{
		AssignmentsClient: client,
	}
}
