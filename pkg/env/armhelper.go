package env

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
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

type ARMHelper interface {
	EnsureARMResourceGroupRoleAssignment(context.Context, refreshable.Authorizer, string) error
}

type noopARMHelper struct{}

func (*noopARMHelper) EnsureARMResourceGroupRoleAssignment(context.Context, refreshable.Authorizer, string) error {
	return nil
}

type armHelper struct {
	log *logrus.Entry
	env Interface

	roleassignments authorization.RoleAssignmentsClient
	applications    graphrbac.ApplicationsClient
}

func newARMHelper(log *logrus.Entry, env Interface) (ARMHelper, error) {
	if os.Getenv("AZURE_ARM_CLIENT_ID") == "" {
		return &noopARMHelper{}, nil
	}

	for _, key := range []string{
		"AZURE_ARM_CLIENT_SECRET",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	ccc := auth.ClientCredentialsConfig{
		ClientID:     os.Getenv("AZURE_ARM_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_ARM_CLIENT_SECRET"),
		TenantID:     env.TenantID(),
		Resource:     env.Environment().ResourceManagerEndpoint,
		AADEndpoint:  env.Environment().ActiveDirectoryEndpoint,
	}
	armAuthorizer, err := ccc.Authorizer()
	if err != nil {
		return nil, err
	}

	fpGraphAuthorizer, err := env.FPAuthorizer(env.TenantID(), env.Environment().GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return &armHelper{
		log: log,
		env: env,

		roleassignments: authorization.NewRoleAssignmentsClient(env.Environment(), env.SubscriptionID(), armAuthorizer),
		applications:    graphrbac.NewApplicationsClient(env.Environment(), env.TenantID(), fpGraphAuthorizer),
	}, nil
}

func (ah *armHelper) EnsureARMResourceGroupRoleAssignment(ctx context.Context, fpAuthorizer refreshable.Authorizer, resourceGroup string) error {
	ah.log.Print("ensuring resource group role assignment")

	res, err := ah.applications.GetServicePrincipalsIDByAppID(ctx, ah.env.FPClientID())
	if err != nil {
		return err
	}

	_, err = ah.roleassignments.Create(ctx, "/subscriptions/"+ah.env.SubscriptionID()+"/resourceGroups/"+resourceGroup, uuid.Must(uuid.NewV4()).String(), mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + ah.env.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/" + rbac.RoleOwner),
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
