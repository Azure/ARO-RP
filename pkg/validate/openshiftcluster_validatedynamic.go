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
	"github.com/Azure/checkaccess-v2-go-sdk/client"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
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
	roleDefinitions armauthorization.RoleDefinitionsClient,
	clusterMsiFederatedIdentityCredentials armmsi.FederatedIdentityCredentialsClient,
	platformWorkloadIdentities map[string]api.PlatformWorkloadIdentity,
	platformWorkloadIdentityRolesByVersion platformworkloadidentity.PlatformWorkloadIdentityRolesByVersion,
	clusterMSICredential azcore.TokenCredential,
) OpenShiftClusterDynamicValidator {
	return &openShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:                                     oc,
		subscriptionDoc:                        subscriptionDoc,
		fpAuthorizer:                           fpAuthorizer,
		roleDefinitions:                        roleDefinitions,
		clusterMsiFederatedIdentityCredentials: clusterMsiFederatedIdentityCredentials,
		platformWorkloadIdentityRolesByVersion: platformWorkloadIdentityRolesByVersion,
		clusterMSICredential:                   clusterMSICredential,
		platformWorkloadIdentities:             platformWorkloadIdentities,
	}
}

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc                                     *api.OpenShiftCluster
	subscriptionDoc                        *api.SubscriptionDocument
	fpAuthorizer                           autorest.Authorizer
	roleDefinitions                        armauthorization.RoleDefinitionsClient
	clusterMsiFederatedIdentityCredentials armmsi.FederatedIdentityCredentialsClient
	platformWorkloadIdentityRolesByVersion platformworkloadidentity.PlatformWorkloadIdentityRolesByVersion
	clusterMSICredential                   azcore.TokenCredential
	platformWorkloadIdentities             map[string]api.PlatformWorkloadIdentity
}

// ensureAccessTokenClaims can detect an error when the service principal (fp, cluster sp) has accidentally deleted from
// the tenant. Using the token with ARM will result in the error as follows
//
//	Lack of an altsecid, puid or oid claim in the token. Continuing would
//	subsequently cause the ARM error `Code="InvalidAuthenticationToken"
//	Message="The received access token is not valid: at least one of the
//	claims 'puid' or 'altsecid' or 'oid' should be present. If you are
//	accessing as an application please make sure service principal is
//	properly created in the tenant."`.  I think this can be returned when
//	the service principal associated with the application hasn't yet
//	caught up with the application itself.
func ensureAccessTokenClaims(ctx context.Context, spTokenCredential azcore.TokenCredential, scopes []string) error {
	options := policy.TokenRequestOptions{Scopes: scopes}
	token, err := spTokenCredential.GetToken(ctx, options)
	if err != nil {
		return err
	}

	var claims jwt.MapClaims
	parser := jwt.NewParser(jwt.WithJSONNumber())
	_, _, err = parser.ParseUnverified(token.Token, &claims)
	if err != nil {
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidServicePrincipalToken,
			"properties.servicePrincipalProfile",
			"The provided service principal generated an invalid token.")
	}

	for _, claim := range []string{"altsecid", "oid", "puid"} {
		if _, found := claims[claim]; found {
			return nil
		}
	}

	return api.NewCloudError(
		http.StatusBadRequest,
		api.CloudErrorCodeInvalidServicePrincipalClaims,
		"properties.servicePrincipalProfile",
		"The Azure Red Hat Openshift resource provider service principal has been removed from your tenant. To restore, please unregister and then re-register the Azure Red Hat OpenShift resource provider.")
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

	tenantID := dv.subscriptionDoc.Subscription.Properties.TenantID
	fpClientCred, err := dv.env.FPNewClientCertificateCredential(tenantID, nil)
	if err != nil {
		return err
	}

	aroEnv := dv.env.Environment()
	clientOptions := &azcore.ClientOptions{
		PerCallPolicies: []policy.Policy{azureclient.NewLoggingPolicy()},
	}
	pdpClient, err := client.NewRemotePDPClient(
		fmt.Sprintf(aroEnv.Endpoint, dv.env.Location()),
		aroEnv.OAuthScope,
		fpClientCred,
		clientOptions,
	)
	if err != nil {
		return err
	}

	scopes := []string{dv.env.Environment().ResourceManagerScope}
	var spDynamic dynamic.Dynamic
	if !dv.oc.UsesWorkloadIdentity() {
		// SP validation
		spp := dv.oc.Properties.ServicePrincipalProfile
		options := dv.env.Environment().ClientSecretCredentialOptions()
		spClientCred, err := azidentity.NewClientSecretCredential(
			tenantID, spp.ClientID, string(spp.ClientSecret), options)
		if err != nil {
			return err
		}
		err = ensureAccessTokenClaims(ctx, spClientCred, scopes)
		if err != nil {
			return err
		}

		spDynamic, err = dynamic.NewValidator(
			dv.log,
			dv.env,
			dv.env.Environment(),
			dv.subscriptionDoc.ID,
			dv.fpAuthorizer,
			&spp.ClientID,
			dynamic.AuthorizerClusterServicePrincipal,
			spClientCred,
			pdpClient,
		)
		if err != nil {
			return err
		}
		err = spDynamic.ValidateServicePrincipal(ctx, spClientCred)
		if err != nil {
			return err
		}
	} else {
		//ClusterMSI Validation
		cmsiDynamic, err := dynamic.NewValidator(
			dv.log,
			dv.env,
			dv.env.Environment(),
			dv.subscriptionDoc.ID,
			dv.fpAuthorizer,
			nil,
			dynamic.AuthorizerClusterUserAssignedIdentity,
			dv.clusterMSICredential,
			pdpClient,
		)
		if err != nil {
			return err
		}
		err = cmsiDynamic.ValidateClusterUserAssignedIdentity(ctx, dv.oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities, dv.roleDefinitions)
		if err != nil {
			return err
		}

		// PlatformWorkloadIdentity Validation
		spDynamic, err = dynamic.NewValidator(
			dv.log,
			dv.env,
			dv.env.Environment(),
			dv.subscriptionDoc.ID,
			dv.fpAuthorizer,
			nil,
			dynamic.AuthorizerWorkloadIdentity,
			fpClientCred,
			pdpClient,
		)
		if err != nil {
			return err
		}
		err = spDynamic.ValidatePlatformWorkloadIdentityProfile(ctx, dv.oc, dv.platformWorkloadIdentityRolesByVersion.GetPlatformWorkloadIdentityRolesByRoleName(), dv.roleDefinitions, dv.clusterMsiFederatedIdentityCredentials, dv.platformWorkloadIdentities)
		if err != nil {
			return err
		}
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

	err = spDynamic.ValidatePreConfiguredNSGs(ctx, dv.oc, subnets)
	if err != nil {
		return err
	}

	err = ensureAccessTokenClaims(ctx, fpClientCred, scopes)
	if err != nil {
		return err
	}

	// FP validation
	fpDynamic, err := dynamic.NewValidator(
		dv.log,
		dv.env,
		dv.env.Environment(),
		dv.subscriptionDoc.ID,
		dv.fpAuthorizer,
		to.StringPtr(dv.env.FPClientID()),
		dynamic.AuthorizerFirstParty,
		fpClientCred,
		pdpClient,
	)
	if err != nil {
		return err
	}

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

	err = fpDynamic.ValidateLoadBalancerProfile(ctx, dv.oc)
	if err != nil {
		return err
	}

	return nil
}
