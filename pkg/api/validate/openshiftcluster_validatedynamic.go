package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate/dynamic"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

// OpenShiftClusterDynamicValidator is the dynamic validator interface
type OpenShiftClusterDynamicValidator interface {
	Dynamic(context.Context) error
}

// NewOpenShiftClusterDynamicValidator creates a new OpenShiftClusterDynamicValidator
func NewOpenShiftClusterDynamicValidator(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster, subscriptionDoc *api.SubscriptionDocument, fpAuthorizer, clusterAuthorizer refreshable.Authorizer, token *adal.ServicePrincipalToken) (*openShiftClusterDynamicValidator, error) {
	//with cluster sp
	providersClusterSP := features.NewProvidersClient(env.Environment(), subscriptionDoc.ID, clusterAuthorizer)
	permissionsClusterSP := authorization.NewPermissionsClient(env.Environment(), subscriptionDoc.ID, clusterAuthorizer)
	diskEncryptionSetsClusterSP := compute.NewDiskEncryptionSetsClient(env.Environment(), subscriptionDoc.ID, clusterAuthorizer)
	resourceSkusClientClusterSP := compute.NewResourceSkusClient(env.Environment(), subscriptionDoc.ID, clusterAuthorizer)
	networkClientClusterSP := network.NewVirtualNetworksClient(env.Environment(), subscriptionDoc.ID, clusterAuthorizer)

	//with first party sp
	permissionsFP := authorization.NewPermissionsClient(env.Environment(), subscriptionDoc.ID, fpAuthorizer)
	diskEncryptionSetsFP := compute.NewDiskEncryptionSetsClient(env.Environment(), subscriptionDoc.ID, fpAuthorizer)
	networkClientFP := network.NewVirtualNetworksClient(env.Environment(), subscriptionDoc.ID, fpAuthorizer)

	tokenClient := aad.NewTokenClient()

	return &openShiftClusterDynamicValidator{
		log: log,
		env: env,

		oc:              oc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,

		tokenClient: tokenClient,

		diskValidator:       dynamic.NewDiskValidator(log, diskEncryptionSetsFP, diskEncryptionSetsClusterSP, permissionsFP, permissionsClusterSP),
		encryptionValidator: dynamic.NewEncryptionAtHostValidator(env, log),
		providersValidator:  dynamic.NewProviderValidator(log, providersClusterSP),
		spValidator:         dynamic.NewServicePrincipalValidator(),
		subnetValidator:     dynamic.NewSubnetValidator(log, networkClientClusterSP),
		vmSKUValidator:      dynamic.NewSKUValidator(resourceSkusClientClusterSP),
		vnetValidator:       dynamic.NewVnetValidator(log, permissionsFP, permissionsClusterSP, networkClientFP, networkClientClusterSP),
	}, nil
}

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    refreshable.Authorizer

	tokenClient aad.TokenClient

	diskValidator       dynamic.DiskValidator
	encryptionValidator dynamic.EncryptionAtHostValidator
	providersValidator  dynamic.ProvidersValidator
	spValidator         dynamic.ServicePrincipalValidator
	subnetValidator     dynamic.SubnetValidator
	vmSKUValidator      dynamic.VMSKUValidator
	vnetValidator       dynamic.VnetValidator
}

func (dv *openShiftClusterDynamicValidator) Dynamic(ctx context.Context) error {
	return dv.dynamic(ctx, dv.env.Environment().ActiveDirectoryEndpoint, dv.env.Environment().ResourceManagerEndpoint)
}

// Dynamic validates an OpenShift cluster
func (dv *openShiftClusterDynamicValidator) dynamic(ctx context.Context, ADEnpoint, RMEndpoint string) error {
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

	err := dv.vnetValidator.Validate(ctx, dv.oc.Location, subnets, dv.oc)
	if err != nil {
		return err
	}

	err = dv.diskValidator.Validate(ctx, dv.oc)
	if err != nil {
		return err
	}

	spp := dv.oc.Properties.ServicePrincipalProfile
	token, err := dv.tokenClient.GetToken(ctx, dv.log, spp.ClientID, string(spp.ClientSecret), dv.subscriptionDoc.Subscription.Properties.TenantID, ADEnpoint, RMEndpoint)
	if err != nil {
		return err
	}

	// validation with cluster service pricinpal
	err = dv.spValidator.Validate(token)
	if err != nil {
		return err
	}

	err = dv.subnetValidator.Validate(ctx, dv.oc, subnets)
	if err != nil {
		return err
	}

	err = dv.providersValidator.Validate(ctx)
	if err != nil {
		return err
	}

	err = dv.encryptionValidator.Validate(ctx, dv.oc)
	if err != nil {
		return err
	}

	err = dv.vmSKUValidator.Validate(ctx, dv.oc.Location, dv.subscriptionDoc.ID, dv.oc)

	if err != nil {
		return err
	}

	return nil
}
