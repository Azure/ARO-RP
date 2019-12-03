package dev

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/env/shared"
)

type dev struct {
	log *logrus.Entry

	*shared.Shared

	roleassignments authorization.RoleAssignmentsClient
	applications    graphrbac.ApplicationsClient
}

type Interface interface {
	CreateARMResourceGroupRoleAssignment(context.Context, autorest.Authorizer, *api.OpenShiftCluster) error
}

func New(ctx context.Context, log *logrus.Entry) (*dev, error) {
	for _, key := range []string{
		"LOCATION",
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	d := &dev{
		log:             log,
		roleassignments: authorization.NewRoleAssignmentsClient(os.Getenv("AZURE_SUBSCRIPTION_ID")),
		applications:    graphrbac.NewApplicationsClient(os.Getenv("AZURE_TENANT_ID")),
	}

	var err error
	d.Shared, err = shared.NewShared(ctx, log, os.Getenv("AZURE_TENANT_ID"), os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"))
	if err != nil {
		return nil, err
	}

	d.roleassignments.Authorizer, err = auth.NewClientCredentialsConfig(os.Getenv("AZURE_ARM_CLIENT_ID"), os.Getenv("AZURE_ARM_CLIENT_SECRET"), os.Getenv("AZURE_TENANT_ID")).Authorizer()
	if err != nil {
		return nil, err
	}

	d.applications.Authorizer, err = d.FPAuthorizer(ctx, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *dev) ListenTLS(ctx context.Context) (net.Listener, error) {
	key, cert, err := d.GetSecret(ctx, "tls")
	if err != nil {
		return nil, err
	}

	// no TLS client cert verification in dev mode, but we'll only listen on
	// localhost
	return tls.Listen("tcp", "localhost:8443", &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					cert.Raw,
				},
				PrivateKey: key,
			},
		},
		MinVersion: tls.VersionTLS12,
	})
}

func (d *dev) Authenticated(h http.Handler) http.Handler {
	return h
}

func (d *dev) IsReady() bool {
	return true
}

func (d *dev) Location() string {
	return os.Getenv("LOCATION")
}

func (d *dev) ResourceGroup() string {
	return os.Getenv("RESOURCEGROUP")
}

func (d *dev) CreateARMResourceGroupRoleAssignment(ctx context.Context, fpAuthorizer autorest.Authorizer, oc *api.OpenShiftCluster) error {
	d.log.Print("development mode: applying resource group role assignment")

	res, err := d.applications.GetServicePrincipalsIDByAppID(ctx, os.Getenv("AZURE_FP_CLIENT_ID"))
	if err != nil {
		return err
	}

	_, err = d.roleassignments.Create(ctx, "/subscriptions/"+os.Getenv("AZURE_SUBSCRIPTION_ID")+"/resourceGroups/"+oc.Properties.ResourceGroup, uuid.NewV4().String(), authorization.RoleAssignmentCreateParameters{
		Properties: &authorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + os.Getenv("AZURE_SUBSCRIPTION_ID") + "/providers/Microsoft.Authorization/roleDefinitions/c95361b8-cf7c-40a1-ad0a-df9f39a30225"),
			PrincipalID:      res.Value,
		},
	})
	if err != nil {
		return err
	}

	fpRefreshableAuthorizer, ok := fpAuthorizer.(*shared.RefreshableAuthorizer)
	if !ok {
		return fmt.Errorf("fpAuthorizer is not refreshable")
	}

	d.log.Print("development mode: refreshing authorizer")
	return fpRefreshableAuthorizer.Refresh()
}
