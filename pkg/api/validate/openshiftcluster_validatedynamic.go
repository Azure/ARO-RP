package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	jwt "github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterDynamicValidator creates a new OpenShiftClusterDynamicValidator
func NewOpenShiftClusterDynamicValidator(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, fpAuthorizer refreshable.Authorizer) OpenShiftClusterDynamicValidator {
	return &openShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:              oc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,
	}
}

type azureClaim struct {
	Roles []string `json:"roles,omitempty"`
}

func (*azureClaim) Valid() error {
	return fmt.Errorf("unimplemented")
}

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    refreshable.Authorizer
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// FP validation
	fpDynamic, err := NewValidator(dv.log, dv.env, dv.oc, dv.subscriptionDoc.ID, dv.fpAuthorizer, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
	if err != nil {
		return err
	}

	err = fpDynamic.ValidateVnetPermissions(ctx)
	if err != nil {
		return err
	}

	err = fpDynamic.ValidateRouteTablesPermissions(ctx)
	if err != nil {
		return err
	}

	// SP validation
	spAuthorizer, err := validateServicePrincipalProfile(ctx, dv.log, dv.env, dv.oc, dv.subscriptionDoc)
	if err != nil {
		return err
	}

	spDynamic, err := NewValidator(dv.log, dv.env, dv.oc, dv.subscriptionDoc.ID, spAuthorizer, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	err = spDynamic.ValidateVnetPermissions(ctx)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateRouteTablesPermissions(ctx)
	if err != nil {
		return err
	}

	err = spDynamic.validateVnet(ctx)
	if err != nil {
		return err
	}

	err = spDynamic.validateProviders(ctx)
	if err != nil {
		return err
	}

	return nil
}

func validateServicePrincipalProfile(ctx context.Context, log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, sub *api.SubscriptionDocument) (refreshable.Authorizer, error) {
	log.Print("validateServicePrincipalProfile")

	token, err := aad.GetToken(ctx, log, oc, sub, env.Environment().ResourceManagerEndpoint)
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

func findSubnet(vnet *mgmtnetwork.VirtualNetwork, subnetID string) *mgmtnetwork.Subnet {
	if vnet.Subnets != nil {
		for _, s := range *vnet.Subnets {
			if strings.EqualFold(*s.ID, subnetID) {
				return &s
			}
		}
	}

	return nil
}

func getRouteTableID(vnet *mgmtnetwork.VirtualNetwork, path, subnetID string) (string, error) {
	s := findSubnet(vnet, subnetID)
	if s == nil {
		return "", api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, path, "The subnet '%s' could not be found.", subnetID)
	}

	if s.RouteTable == nil {
		return "", nil
	}

	return *s.RouteTable.ID, nil
}
