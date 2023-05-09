package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/form3tech-oss/jwt-go"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/permissions"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewFirstPartyOpenShiftClusterDynamicValidator creates a new
// OpenShiftClusterDynamicValidator that operates using the first party service
// principal (FPSP)
func NewFirstPartyOpenShiftClusterDynamicValidator(
	log *logrus.Entry,
	env env.Interface,
	oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument,
	fpAuthorizer autorest.Authorizer,
) OpenShiftClusterDynamicValidator {
	return &firstPartyOpenShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:              oc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,
	}
}

// NewClientOpenShiftClusterDynamicValidator creates a new
// OpenShiftClusterDynamicValidator that operates using the client service
// principal
func NewClientOpenShiftClusterDynamicValidator(
	log *logrus.Entry,
	env env.Interface,
	oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument,
) OpenShiftClusterDynamicValidator {
	return &clientOpenShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:              oc,
		subscriptionDoc: subscriptionDoc,
	}
}

type firstPartyOpenShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    autorest.Authorizer
}

type clientOpenShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
}

func ensureAccessTokenClaims(ctx context.Context, tokenCredential *azidentity.ClientSecretCredential, scopes []string) error {
	var err error

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the
	// latest error to the user in case the wait exceeds the timeout.
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		options := policy.TokenRequestOptions{Scopes: scopes}
		token, err := tokenCredential.GetToken(ctx, options)
		if err != nil {
			return false, err
		}

		var claims jwt.MapClaims
		parser := &jwt.Parser{UseJSONNumber: true}
		_, _, err = parser.ParseUnverified(token.Token, &claims)
		if err != nil {
			err = api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidServicePrincipalToken,
				"properties.servicePrincipalProfile",
				"The provided service principal generated an invalid token.")
			return false, err
		}

		// XXX Unclear if this check is still required, as it was originally
		//     implemented for ADAL authentication with the comment:
		//
		//     Lack of an altsecid, puid or oid claim in the token. Continuing would
		//     subsequently cause the ARM error `Code="InvalidAuthenticationToken"
		//     Message="The received access token is not valid: at least one of the
		//     claims 'puid' or 'altsecid' or 'oid' should be present. If you are
		//     accessing as an application please make sure service principal is
		//     properly created in the tenant."`.  I think this can be returned when
		//     the service principal associated with the application hasn't yet
		//     caught up with the application itself.
		//
		//     (source: commit id 52dff30f31bad63cc4e46bbf701437756a6da83a)
		for _, claim := range []string{"altsecid", "oid", "puid"} {
			if _, found := claims[claim]; found {
				return true, nil
			}
		}

		err = api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidServicePrincipalClaims,
			"properties.servicePrincipleProfile",
			"The provided service principal does not give an access token with at least one of the claims 'altsecid', 'oid' or 'puid'.")
		return false, err
	}, timeoutCtx.Done())

	return err
}

// Dynamic validates an OpenShift cluster
func (dv *firstPartyOpenShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// Get all subnets
	subnets := []dynamic.Subnet{{
		ID:   dv.oc.Properties.MasterProfile.SubnetID,
		Path: "properties.masterProfile.subnetId",
	}}
	for i, wp := range dv.oc.Properties.WorkerProfiles {
		subnets = append(subnets, dynamic.Subnet{
			ID:   wp.SubnetID,
			Path: fmt.Sprintf("properties.workerProfiles[%d].subnetId", i),
		})
	}

	auth, err := permissions.NewPermissionsValidator(dv.env, dv.log, dv.fpAuthorizer, dv.oc, dv.subscriptionDoc)
	if err != nil {
		return err
	}

	fpDynamic := dynamic.NewValidator(
		dv.log,
		dv.env,
		dv.env.Environment(),
		dv.subscriptionDoc.ID,
		dv.fpAuthorizer,
		dv.env.FPClientID(),
		dynamic.AuthorizerFirstParty,
		auth,
	)

	err = fpDynamic.ValidateDiskEncryptionSets(ctx, dv.oc)
	if err != nil {
		return err
	}

	fpDynamicVnet := dynamic.NewVirtualNetworkValidator(
		dv.log,
		dv.env,
		dv.env.Environment(),
		dv.subscriptionDoc.ID,
		dv.fpAuthorizer,
		dv.env.FPClientID(),
		dynamic.AuthorizerFirstParty,
		auth,
	)

	err = fpDynamicVnet.ValidateVnet(
		ctx,
		dv.oc.Location,
		subnets,
		dv.oc.Properties.NetworkProfile.PodCIDR,
		dv.oc.Properties.NetworkProfile.ServiceCIDR,
	)
	if err != nil {
		return err
	}

	return nil
}

// Dynamic validates an OpenShift cluster
func (dv *clientOpenShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// Get all subnets
	subnets := []dynamic.Subnet{{
		ID:   dv.oc.Properties.MasterProfile.SubnetID,
		Path: "properties.masterProfile.subnetId",
	}}
	for i, wp := range dv.oc.Properties.WorkerProfiles {
		subnets = append(subnets, dynamic.Subnet{
			ID:   wp.SubnetID,
			Path: fmt.Sprintf("properties.workerProfiles[%d].subnetId", i),
		})
	}

	tokenCredential, err := clusterauthorizer.NewTokenCredentialForCluster(
		dv.env.Environment(), dv.oc, dv.subscriptionDoc.Subscription)
	if err != nil {
		return err
	}

	scopes := []string{dv.env.Environment().ResourceManagerScope}
	err = ensureAccessTokenClaims(ctx, tokenCredential, scopes)
	if err != nil {
		return err
	}
	spAuthorizer := azidext.NewTokenCredentialAdapter(tokenCredential, scopes)

	auth, err := permissions.NewPermissionsValidator(dv.env, dv.log, spAuthorizer, dv.oc, dv.subscriptionDoc)
	if err != nil {
		return err
	}

	spValidator := dynamic.NewServicePrincipalValidator(
		dv.log, dv.env.Environment(), dynamic.AuthorizerClusterServicePrincipal,
	)

	// SP validation
	err = spValidator.ValidateServicePrincipal(ctx, tokenCredential)
	if err != nil {
		return err
	}

	spDynamic := dynamic.NewValidator(
		dv.log,
		dv.env,
		dv.env.Environment(),
		dv.subscriptionDoc.ID,
		spAuthorizer,
		dv.oc.Properties.ServicePrincipalProfile.ClientID,
		dynamic.AuthorizerClusterServicePrincipal,
		auth,
	)

	err = spDynamic.ValidateDiskEncryptionSets(ctx, dv.oc)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateEncryptionAtHost(ctx, dv.oc)
	if err != nil {
		return err
	}

	spNetworkDynamic := dynamic.NewVirtualNetworkValidator(
		dv.log,
		dv.env,
		dv.env.Environment(),
		dv.subscriptionDoc.ID,
		spAuthorizer,
		dv.oc.Properties.ServicePrincipalProfile.ClientID,
		dynamic.AuthorizerClusterServicePrincipal,
		auth,
	)

	err = spNetworkDynamic.ValidateVnet(
		ctx,
		dv.oc.Location,
		subnets,
		dv.oc.Properties.NetworkProfile.PodCIDR,
		dv.oc.Properties.NetworkProfile.ServiceCIDR,
	)
	if err != nil {
		return err
	}

	err = spNetworkDynamic.ValidateSubnets(ctx, dv.oc, subnets)
	if err != nil {
		return err
	}

	return nil
}
