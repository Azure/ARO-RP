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

type openShiftClusterDynamicValidator struct {
	log *logrus.Entry
	env env.Interface

	oc              *api.OpenShiftCluster
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    refreshable.Authorizer
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
	fpDynamic, err := dynamic.NewValidator(dv.log, dv.env, dv.env.Environment(), dv.subscriptionDoc.ID, dv.fpAuthorizer, dynamic.AuthorizerFirstParty)
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

	spp := dv.oc.Properties.ServicePrincipalProfile
	token, err := aad.GetToken(ctx, dv.log, spp.ClientID, string(spp.ClientSecret), dv.subscriptionDoc.Subscription.Properties.TenantID, dv.env.Environment().ActiveDirectoryEndpoint, dv.env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	spAuthorizer := refreshable.NewAuthorizer(token)

	spDynamic, err := dynamic.NewValidator(dv.log, dv.env, dv.env.Environment(), dv.subscriptionDoc.ID, spAuthorizer, dynamic.AuthorizerClusterServicePrincipal)
	if err != nil {
		return err
	}

	// SP validation
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

	err = spDynamic.ValidateQuota(ctx, dv.oc)
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

	err = spDynamic.ValidateVMSku(ctx, dv.oc.Location, dv.subscriptionDoc.ID, dv.oc)
	if err != nil {
		return err
	}

	return nil
}
