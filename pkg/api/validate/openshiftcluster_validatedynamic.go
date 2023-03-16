package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/authz/remotepdp"
	"github.com/Azure/ARO-RP/pkg/util/feature"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterDynamicValidator creates a new OpenShiftClusterDynamicValidator
func NewOpenShiftClusterDynamicValidator(ctx context.Context, log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, fpAuthorizer refreshable.Authorizer) (OpenShiftClusterDynamicValidator, error) {
	var fpClientCred azcore.TokenCredential
	var spClientCred azcore.TokenCredential
	var pdpClient remotepdp.RemotePDPClient

	spp := oc.Properties.ServicePrincipalProfile

	if feature.IsRegisteredForFeature(subscriptionDoc.Subscription.Properties, api.FeatureFlagCheckAccessTestToggle) {
		log.Info("CheckAccess Feature is set")
		var err error
		fpClientCred, err = env.FPNewClientCertificateCredential(subscriptionDoc.Subscription.Properties.TenantID)
		if err != nil {
			return nil, err
		}

		spClientCred, err = azidentity.NewClientSecretCredential(subscriptionDoc.Subscription.Properties.TenantID, spp.ClientID, string(spp.ClientSecret), nil)
		if err != nil {
			return nil, err
		}

		aroEnv := env.Environment()
		pdpClient = remotepdp.NewRemotePDPClient(
			aroEnv.AzureRbacPDPEnvironment.Endpoint,
			aroEnv.AzureRbacPDPEnvironment.OAuthScope,
			fpClientCred,
		)
	}

	tokenClient := aad.NewTokenClient()
	token, err := tokenClient.GetToken(ctx, log, spp.ClientID, string(spp.ClientSecret), subscriptionDoc.Subscription.Properties.TenantID, env.Environment().ActiveDirectoryEndpoint, env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}
	spAuthorizer := refreshable.NewAuthorizer(token)

	return &openShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:                 oc,
		subscriptionDoc:    subscriptionDoc,
		fpAuthorizer:       fpAuthorizer,
		spAuthorizer:       spAuthorizer,
		fpClientCredential: fpClientCred,
		spClientCredential: spClientCred,
		pdpClient:          pdpClient,
	}, nil
}

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc                 *api.OpenShiftCluster
	subscriptionDoc    *api.SubscriptionDocument
	fpAuthorizer       refreshable.Authorizer
	spAuthorizer       refreshable.Authorizer
	fpClientCredential azcore.TokenCredential
	spClientCredential azcore.TokenCredential
	pdpClient          remotepdp.RemotePDPClient
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
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

	// FP validation
	fpDynamic, err := dynamic.NewValidator(dv.log, dv.env, dv.env.Environment(), dv.subscriptionDoc.ID, dv.fpAuthorizer, dynamic.AuthorizerFirstParty, aad.NewTokenClient(), dv.pdpClient)
	if err != nil {
		return err
	}

	err = fpDynamic.ValidateVnet(ctx, dv.oc.Location, subnets, dv.oc.Properties.NetworkProfile.PodCIDR, dv.oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		return err
	}

	err = fpDynamic.ValidateDiskEncryptionSets(ctx, dv.oc)
	if err != nil {
		return err
	}

	//TODO create the cluster SP and add to NewValidator
	spDynamic, err := dynamic.NewValidator(dv.log, dv.env, dv.env.Environment(), dv.subscriptionDoc.ID, dv.spAuthorizer, dynamic.AuthorizerClusterServicePrincipal, aad.NewTokenClient(), dv.pdpClient)
	if err != nil {
		return err
	}

	// SP validation
	spp := dv.oc.Properties.ServicePrincipalProfile
	err = spDynamic.ValidateServicePrincipal(ctx, spp.ClientID, string(spp.ClientSecret), dv.subscriptionDoc.Subscription.Properties.TenantID)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateVnet(ctx, dv.oc.Location, subnets, dv.oc.Properties.NetworkProfile.PodCIDR, dv.oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateSubnets(ctx, dv.oc, subnets)
	if err != nil {
		return err
	}

	err = spDynamic.ValidateProviders(ctx)
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

	return nil
}
