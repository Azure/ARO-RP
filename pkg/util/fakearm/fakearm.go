package fakearm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
)

type FakeARM interface {
	CreateARMResourceGroupRoleAssignment(ctx context.Context, log *logrus.Entry, _env env.Interface, resourceGroup string) error
}

func New(_env env.Interface, fp env.FPAuthorizer) (FakeARM, error) {
	if _env.Type() != env.Dev {
		return &noop{}, nil
	}

	for _, key := range []string{
		"AZURE_ARM_CLIENT_ID",
		"AZURE_ARM_CLIENT_SECRET",
		"AZURE_FP_CLIENT_ID",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset (development mode)", key)
		}
	}

	fpGraphAuthorizer, err := fp.FPAuthorizer(_env.TenantID(), azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	armAuthorizer, err := auth.NewClientCredentialsConfig(os.Getenv("AZURE_ARM_CLIENT_ID"), os.Getenv("AZURE_ARM_CLIENT_SECRET"), _env.TenantID()).Authorizer()
	if err != nil {
		return nil, err
	}

	return &fakearm{
		applications:    graphrbac.NewApplicationsClient(_env.TenantID(), fpGraphAuthorizer),
		roleassignments: authorization.NewRoleAssignmentsClient(_env.SubscriptionID(), armAuthorizer),
	}, nil
}

type noop struct{}

func (noop) CreateARMResourceGroupRoleAssignment(context.Context, *logrus.Entry, env.Interface, string) error {
	return nil
}

type fakearm struct {
	applications    graphrbac.ApplicationsClient
	roleassignments authorization.RoleAssignmentsClient
}

func (f *fakearm) CreateARMResourceGroupRoleAssignment(ctx context.Context, log *logrus.Entry, _env env.Interface, resourceGroup string) error {
	log.Print("development mode: applying resource group role assignment")

	res, err := f.applications.GetServicePrincipalsIDByAppID(ctx, os.Getenv("AZURE_FP_CLIENT_ID"))
	if err != nil {
		return err
	}

	_, err = f.roleassignments.Create(ctx, "/subscriptions/"+_env.SubscriptionID()+"/resourceGroups/"+resourceGroup, uuid.NewV4().String(), mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + _env.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635"),
			PrincipalID:      res.Value,
		},
	})
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "RoleAssignmentExists" {
			err = nil
		}
	}

	return err
}
