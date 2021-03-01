package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/form3tech-oss/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclaim"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterDynamicValidator creates a new OpenShiftClusterDynamicValidator
func NewOpenShiftClusterDynamicValidator(log *logrus.Entry, env env.Core, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, fpAuthorizer refreshable.Authorizer) OpenShiftClusterDynamicValidator {
	return &openShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:              oc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,
	}
}

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Core

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    refreshable.Authorizer
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// Get all subnets
	mSubnetID := dv.oc.Properties.MasterProfile.SubnetID
	wSubnetIDs := []string{}

	for _, s := range dv.oc.Properties.WorkerProfiles {
		wSubnetIDs = append(wSubnetIDs, s.SubnetID)
	}

	// FP validation
	fpDynamic, err := dynamic.NewValidator(dv.log, dv.env, dv.oc, dv.subscriptionDoc, mSubnetID, wSubnetIDs, dv.subscriptionDoc.ID, dv.fpAuthorizer, api.CloudErrorCodeInvalidResourceProviderPermissions, "resource provider")
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

	spp := dv.oc.Properties.ServicePrincipalProfile
	token, err := aad.GetToken(ctx, dv.log, spp.ClientID, string(spp.ClientSecret), dv.subscriptionDoc.Subscription.Properties.TenantID, dv.env.Environment().ActiveDirectoryEndpoint, dv.env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	spAuthorizer := refreshable.NewAuthorizer(token)

	spDynamic, err := dynamic.NewValidator(dv.log, dv.env, dv.oc, dv.subscriptionDoc, mSubnetID, wSubnetIDs, dv.subscriptionDoc.ID, spAuthorizer, api.CloudErrorCodeInvalidServicePrincipalPermissions, "provided service principal")
	if err != nil {
		return err
	}

	// SP validation
	err = spDynamic.ValidateClusterServicePrincipalProfile(ctx)
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

	// Additional checks - use any dynamic because they both have the correct permissions
	err = spDynamic.ValidateVnetLocation(ctx)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateProviders(ctx)
	if err != nil {
		return err
	}

	return nil
}

func validateServicePrincipalProfile(ctx context.Context, log *logrus.Entry, env env.Core, oc *api.OpenShiftCluster, sub *api.SubscriptionDocument) error {
	// TODO: once aad.GetToken is mockable, write a unit test for this function

	log.Print("validateServicePrincipalProfile")

	spp := oc.Properties.ServicePrincipalProfile
	token, err := aad.GetToken(ctx, log, spp.ClientID, string(spp.ClientSecret), sub.Subscription.Properties.TenantID, env.Environment().ActiveDirectoryEndpoint, env.Environment().GraphEndpoint)
	if err != nil {
		return err
	}

	p := &jwt.Parser{}
	c := &azureclaim.AzureClaim{}
	_, _, err = p.ParseUnverified(token.OAuthToken(), c)
	if err != nil {
		return err
	}

	for _, role := range c.Roles {
		if role == "Application.ReadWrite.OwnedBy" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidServicePrincipalCredentials, "properties.servicePrincipalProfile", "The provided service principal must not have the Application.ReadWrite.OwnedBy permission.")
		}
	}

	return nil
}
