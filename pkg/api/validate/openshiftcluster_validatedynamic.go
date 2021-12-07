package validate

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

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
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Failed to validate the first principal",
		)
	}

	err = fpDynamic.ValidateVnet(ctx, dv.oc.Location, subnets, dv.oc.Properties.NetworkProfile.PodCIDR, dv.oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedVNet,
			"", "Invalid virtual network specified",
		)
	}

	err = fpDynamic.ValidateDiskEncryptionSets(ctx, dv.oc)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Invalid disk encryption sets specified",
		)
	}

	spp := dv.oc.Properties.ServicePrincipalProfile
	token, err := aad.GetToken(ctx, dv.log, spp.ClientID, string(spp.ClientSecret), dv.subscriptionDoc.Subscription.Properties.TenantID, dv.env.Environment().ActiveDirectoryEndpoint, dv.env.Environment().ResourceManagerEndpoint)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Failed to retrieve an AAD token for the service principal",
		)
	}

	spAuthorizer := refreshable.NewAuthorizer(token)

	spDynamic, err := dynamic.NewValidator(dv.log, dv.env, dv.env.Environment(), dv.subscriptionDoc.ID, spAuthorizer, dynamic.AuthorizerClusterServicePrincipal)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Failed to validate the service principal token",
		)
	}

	// SP validation
	err = spDynamic.ValidateServicePrincipal(ctx, spp.ClientID, string(spp.ClientSecret), dv.subscriptionDoc.Subscription.Properties.TenantID)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Failed to validate the cluster service principal",
		)
	}

	err = spDynamic.ValidateVnet(ctx, dv.oc.Location, subnets, dv.oc.Properties.NetworkProfile.PodCIDR, dv.oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedVNet,
			"", "Invalid cluster virtual network CIDRs",
		)
	}

	err = spDynamic.ValidateSubnets(ctx, dv.oc, subnets)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Invalid cluster virtual network subnets",
		)
	}

	err = spDynamic.ValidateProviders(ctx)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Failed to validate the registered resource providers",
		)
	}

	err = spDynamic.ValidateQuota(ctx, dv.oc)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeQuotaExceeded,
			"", "Current resource quotas do not permit the creation of the cluster",
		)
	}

	err = spDynamic.ValidateDiskEncryptionSets(ctx, dv.oc)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidLinkedDiskEncryptionSet,
			"", "Invalid disk encryption sets for the specified service principal",
		)
	}

	err = spDynamic.ValidateEncryptionAtHost(ctx, dv.oc)
	if err != nil {
		if _, isCloudError := err.(*api.CloudError); isCloudError {
			return err
		}
		dv.log.Error(err)
		return api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"", "Invalid host encryption for the specified service principal",
		)
	}

	return nil
}
