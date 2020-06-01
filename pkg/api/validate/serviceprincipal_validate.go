package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	utilpermissions "github.com/Azure/ARO-RP/pkg/util/permissions"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// ServicePrincipalValidator validates that the SPP has the correct permissions
type ServicePrincipalValidator interface {
	Validate(context.Context) error
	Authorizer() refreshable.Authorizer
}

// NewServicePrincipalValidator creates a new ServicePrincipalValidator
func NewServicePrincipalValidator(log *logrus.Entry, spp *api.ServicePrincipalProfile, clusterID, masterSubnetID, workerSubnetID string) ServicePrincipalValidator {
	return &servicePrincipalValidator{
		log: log,

		spp:            spp,
		clusterID:      clusterID,
		masterSubnetID: masterSubnetID,
		workerSubnetID: workerSubnetID,
	}
}

type azureClaim struct {
	Roles []string `json:"roles,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}

type servicePrincipalValidator struct {
	log *logrus.Entry

	spp            *api.ServicePrincipalProfile
	clusterID      string
	masterSubnetID string
	workerSubnetID string

	spPermissions     authorization.PermissionsClient
	authorizer        refreshable.Authorizer
	spVirtualNetworks network.VirtualNetworksClient
}

// validates a service principal
func (dv *servicePrincipalValidator) Validate(ctx context.Context) error {
	r, err := azure.ParseResourceID(dv.clusterID)
	if err != nil {
		return err
	}

	dv.authorizer, err = dv.validateServicePrincipalProfile(ctx)
	if err != nil {
		return err
	}

	dv.spPermissions = authorization.NewPermissionsClient(r.SubscriptionID, dv.authorizer)
	dv.spVirtualNetworks = network.NewVirtualNetworksClient(r.SubscriptionID, dv.authorizer)

	vnetID, _, err := subnet.Split(dv.masterSubnetID)
	if err != nil {
		return err
	}

	vnetr, err := azure.ParseResourceID(vnetID)
	if err != nil {
		return err
	}

	err = validateVnetPermissions(ctx, dv.log, dv.authorizer, dv.spPermissions, vnetID, &vnetr, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	// Get after validating permissions
	vnet, err := dv.spVirtualNetworks.Get(ctx, vnetr.ResourceGroup, vnetr.ResourceName, "")
	if err != nil {
		return err
	}

	return dv.validateRouteTablePermissions(ctx, dv.spPermissions, &vnet, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
}

func (dv *servicePrincipalValidator) Authorizer() refreshable.Authorizer {
	return dv.authorizer
}

func (dv *servicePrincipalValidator) validateServicePrincipalProfile(ctx context.Context) (refreshable.Authorizer, error) {
	dv.log.Print("validateServicePrincipalProfile")

	token, err := aad.GetToken(ctx, dv.log, dv.spp, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	p := &jwt.Parser{}
	c := &azureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return nil, err
	}

	for _, role := range c.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}
	return refreshable.NewAuthorizer(token), nil
}

func validateVnetPermissions(ctx context.Context, log *logrus.Entry, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnetID string, vnetr *azure.Resource, code, typ string) error {
	log.Printf("validateVnetPermissions (%s)", typ)

	err := validateActions(ctx, log, vnetr, []string{
		"Microsoft.Network/virtualNetworks/subnets/join/action",
		"Microsoft.Network/virtualNetworks/subnets/read",
		"Microsoft.Network/virtualNetworks/subnets/write",
	}, authorizer, client)
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Contributor permission on vnet '%s'.", typ, vnetID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The vnet '%s' could not be found.", vnetID)
	}
	return err
}

func (dv *servicePrincipalValidator) validateRouteTablePermissions(ctx context.Context, client authorization.PermissionsClient, vnet *mgmtnetwork.VirtualNetwork, code, typ string) error {
	err := validateRouteTablePermissionsSubnet(ctx, dv.log, dv.authorizer, client, vnet, dv.masterSubnetID, "properties.masterProfile.subnetId", code, typ)
	if err != nil {
		return err
	}

	return validateRouteTablePermissionsSubnet(ctx, dv.log, dv.authorizer, client, vnet, dv.workerSubnetID, `properties.workerProfiles["worker"].subnetId`, code, typ)
}

func validateRouteTablePermissionsSubnet(ctx context.Context, log *logrus.Entry, authorizer refreshable.Authorizer, client authorization.PermissionsClient, vnet *mgmtnetwork.VirtualNetwork, subnetID, path, code, typ string) error {
	log.Printf("validateRouteTablePermissionsSubnet(%s, %s)", typ, path)

	var s *mgmtnetwork.Subnet
	for _, ss := range *vnet.Subnets {
		if strings.EqualFold(*ss.ID, subnetID) {
			s = &ss
			break
		}
	}
	if s == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The subnet '%s' could not be found.", subnetID)
	}

	if s.RouteTable == nil {
		return nil
	}

	rtr, err := azure.ParseResourceID(*s.RouteTable.ID)
	if err != nil {
		return err
	}

	err = validateActions(ctx, log, &rtr, []string{
		"Microsoft.Network/routeTables/join/action",
		"Microsoft.Network/routeTables/read",
		"Microsoft.Network/routeTables/write",
	}, authorizer, client)
	if err == wait.ErrWaitTimeout {
		return api.NewCloudError(http.StatusBadRequest, code, "", "The %s does not have Contributor permission on route table '%s'.", typ, *s.RouteTable.ID)
	}
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedRouteTable, "", "The route table '%s' could not be found.", *s.RouteTable.ID)
	}
	return err
}

func validateActions(ctx context.Context, log *logrus.Entry, r *azure.Resource, actions []string, authorizer refreshable.Authorizer, client authorization.PermissionsClient) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	return wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		permissions, err := client.ListForResource(ctx, r.ResourceGroup, r.Provider, "", r.ResourceType, r.ResourceName)
		if detailedErr, ok := err.(autorest.DetailedError); ok &&
			detailedErr.StatusCode == http.StatusForbidden {
			log.Print(err)
			err = authorizer.RefreshWithContext(ctx)
			if err != nil {
				return false, err
			}
			return false, nil
		}
		if err != nil {
			return false, err
		}

		for _, action := range actions {
			ok, err := utilpermissions.CanDoAction(permissions, action)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}

		return true, nil
	}, timeoutCtx.Done())
}
