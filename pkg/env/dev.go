package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net"
	"os"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type dev struct {
	*prod

	permissions     authorization.PermissionsClient
	roleassignments authorization.RoleAssignmentsClient
	applications    graphrbac.ApplicationsClient
	deployments     features.DeploymentsClient
}

func newDev(ctx context.Context, log *logrus.Entry) (Interface, error) {
	for _, key := range []string{
		"AZURE_ARM_CLIENT_ID",
		"AZURE_ARM_CLIENT_SECRET",
		"PROXY_HOSTNAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	d := &dev{}

	var err error
	d.prod, err = newProd(ctx, log)
	if err != nil {
		return nil, err
	}

	d.features[FeatureDisableDenyAssignments] = true
	d.features[FeatureDisableSignedCertificates] = true

	ccc := auth.ClientCredentialsConfig{
		ClientID:     os.Getenv("AZURE_ARM_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_ARM_CLIENT_SECRET"),
		TenantID:     d.TenantID(),
		Resource:     d.Environment().ResourceManagerEndpoint,
		AADEndpoint:  d.Environment().ActiveDirectoryEndpoint,
	}
	armAuthorizer, err := ccc.Authorizer()
	if err != nil {
		return nil, err
	}

	d.roleassignments = authorization.NewRoleAssignmentsClient(d.Environment(), d.SubscriptionID(), armAuthorizer)
	d.prod.clusterGenevaLoggingEnvironment = "Test"
	d.prod.clusterGenevaLoggingConfigVersion = "2.3"

	fpGraphAuthorizer, err := d.FPAuthorizer(d.TenantID(), d.Environment().GraphEndpoint)
	if err != nil {
		return nil, err
	}

	d.applications = graphrbac.NewApplicationsClient(d.Environment(), d.TenantID(), fpGraphAuthorizer)

	fpAuthorizer, err := d.FPAuthorizer(d.TenantID(), d.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	d.permissions = authorization.NewPermissionsClient(d.Environment(), d.SubscriptionID(), fpAuthorizer)

	d.deployments = features.NewDeploymentsClient(d.Environment(), d.TenantID(), fpAuthorizer)

	return d, nil
}

func (d *dev) InitializeAuthorizers() error {
	d.armClientAuthorizer = clientauthorizer.NewAll()
	d.adminClientAuthorizer = clientauthorizer.NewAll()
	return nil
}

func (d *dev) AROOperatorImage() string {
	override := os.Getenv("ARO_IMAGE")
	if override != "" {
		return override
	}

	return fmt.Sprintf("%s/aro:%s", d.ACRDomain(), version.GitCommit)
}

func (d *dev) Listen() (net.Listener, error) {
	// in dev mode there is no authentication, so for safety we only listen on
	// localhost
	return net.Listen("tcp", "localhost:8443")
}

func (d *dev) FPAuthorizer(tenantID, resource string) (refreshable.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(d.Environment().ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, d.fpClientID, d.fpCertificate, d.fpPrivateKey, resource)
	if err != nil {
		return nil, err
	}

	return refreshable.NewAuthorizer(sp), nil
}

func (d *dev) EnsureARMResourceGroupRoleAssignment(ctx context.Context, fpAuthorizer refreshable.Authorizer, resourceGroup string) error {
	d.log.Print("development mode: ensuring resource group role assignment")

	res, err := d.applications.GetServicePrincipalsIDByAppID(ctx, d.fpClientID)
	if err != nil {
		return err
	}

	_, err = d.roleassignments.Create(ctx, "/subscriptions/"+d.SubscriptionID()+"/resourceGroups/"+resourceGroup, uuid.Must(uuid.NewV4()).String(), mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + d.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/" + rbac.RoleOwner),
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
