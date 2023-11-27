package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/form3tech-oss/jwt-go"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/feature"
	"github.com/Azure/ARO-RP/pkg/validate/dynamic"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterDynamicValidator creates a new OpenShiftClusterDynamicValidator
func NewOpenShiftClusterDynamicValidator(
	log *logrus.Entry,
	env env.Interface,
	oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument,
	fpAuthorizer autorest.Authorizer,
) OpenShiftClusterDynamicValidator {
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
	env env.Interface

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    autorest.Authorizer
}

func ensureAccessTokenClaims(ctx context.Context, spTokenCredential azcore.TokenCredential, scopes []string) error {
	options := policy.TokenRequestOptions{Scopes: scopes}
	token, err := spTokenCredential.GetToken(ctx, options)
	if err != nil {
		return err
	}

	var claims jwt.MapClaims
	parser := &jwt.Parser{UseJSONNumber: true}
	_, _, err = parser.ParseUnverified(token.Token, &claims)
	if err != nil {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidServicePrincipalToken,
			"properties.servicePrincipalProfile",
			"The provided service principal generated an invalid token.")
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
			return nil
		}
	}

	return api.NewCloudError(
		http.StatusBadRequest,
		api.CloudErrorCodeInvalidServicePrincipalClaims,
		"properties.servicePrincipleProfile",
		"The provided service principal does not give an access token with at least one of the claims 'altsecid', 'oid' or 'puid'.")
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	// Get all subnets
	subnets := []dynamic.Subnet{{
		ID:   dv.oc.Properties.MasterProfile.SubnetID,
		Path: "properties.masterProfile.subnetId",
	}}

	workerProfiles, propertyName := api.GetEnrichedWorkerProfiles(dv.oc.Properties)
	for i, wp := range workerProfiles {
		subnets = append(subnets, dynamic.Subnet{
			ID:   wp.SubnetID,
			Path: fmt.Sprintf("properties.%s[%d].subnetId", propertyName, i),
		})
	}

	var fpClientCred azcore.TokenCredential
	var spClientCred azcore.TokenCredential
	var pdpClient remotepdp.RemotePDPClient
	spp := dv.oc.Properties.ServicePrincipalProfile

	useCheckAccess, err := dv.env.LiveConfig().UseCheckAccess(ctx)
	dv.log.Info("USE_CHECKACCESS: ", useCheckAccess)
	if err != nil {
		return err
	}

	if useCheckAccess || feature.IsRegisteredForFeature(
		dv.subscriptionDoc.Subscription.Properties,
		api.FeatureFlagCheckAccessTestToggle,
	) {
		// TODO remove after successfully migrating to CheckAccess
		dv.log.Info("Using CheckAccess instead of ListPermissions")
		var err error
		fpClientCred, err = dv.env.FPNewClientCertificateCredential(dv.subscriptionDoc.Subscription.Properties.TenantID)
		if err != nil {
			return err
		}

		spClientCred, err = azidentity.NewClientSecretCredential(
			dv.subscriptionDoc.Subscription.Properties.TenantID,
			spp.ClientID,
			string(spp.ClientSecret),
			nil,
		)
		if err != nil {
			return err
		}

		aroEnv := dv.env.Environment()
		pdpClient = remotepdp.NewRemotePDPClient(
			fmt.Sprintf(aroEnv.Endpoint, dv.env.Location()),
			aroEnv.OAuthScope,
			fpClientCred,
		)
	}

	tenantID := dv.subscriptionDoc.Subscription.Properties.TenantID
	options := dv.env.Environment().ClientSecretCredentialOptions()
	spTokenCredential, err := azidentity.NewClientSecretCredential(
		tenantID, spp.ClientID, string(spp.ClientSecret), options)
	if err != nil {
		return err
	}

	scopes := []string{dv.env.Environment().ResourceManagerScope}
	err = ensureAccessTokenClaims(ctx, spTokenCredential, scopes)
	if err != nil {
		return err
	}
	spAuthorizer := azidext.NewTokenCredentialAdapter(spTokenCredential, scopes)

	spDynamic := dynamic.NewValidator(
		dv.log,
		dv.env,
		dv.env.Environment(),
		dv.subscriptionDoc.ID,
		spAuthorizer,
		spp.ClientID,
		dynamic.AuthorizerClusterServicePrincipal,
		spClientCred,
		pdpClient,
	)

	// SP validation
	err = spDynamic.ValidateServicePrincipal(ctx, spTokenCredential)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateVnet(
		ctx,
		dv.oc.Location,
		subnets,
		dv.oc.Properties.NetworkProfile.PodCIDR,
		dv.oc.Properties.NetworkProfile.ServiceCIDR,
	)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateSubnets(ctx, dv.oc, subnets)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateDiskEncryptionSets(ctx, dv.oc)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateEncryptionAtHost(ctx, dv.oc)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateLoadBalancerProfile(ctx, dv.oc)
	if err != nil {
		return err
	}

	err = spDynamic.ValidatePreConfiguredNSGs(ctx, dv.oc, subnets)
	if err != nil {
		return err
	}

	// FP validation
	fpDynamic := dynamic.NewValidator(
		dv.log,
		dv.env,
		dv.env.Environment(),
		dv.subscriptionDoc.ID,
		dv.fpAuthorizer,
		dv.env.FPClientID(),
		dynamic.AuthorizerFirstParty,
		fpClientCred,
		pdpClient,
	)

	err = fpDynamic.ValidateVnet(
		ctx,
		dv.oc.Location,
		subnets,
		dv.oc.Properties.NetworkProfile.PodCIDR,
		dv.oc.Properties.NetworkProfile.ServiceCIDR,
	)
	if err != nil {
		return err
	}

	err = fpDynamic.ValidateDiskEncryptionSets(ctx, dv.oc)
	if err != nil {
		return err
	}

	err = fpDynamic.ValidatePreConfiguredNSGs(ctx, dv.oc, subnets)
	if err != nil {
		return err
	}

	return nil
}
