package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

// NOTE: Make sure to change and always return 200, with any errors inside of the response message
func (f *frontend) preflightValidation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	var header http.Header
	var b []byte

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	originalPath := r.Context().Value(middleware.ContextKeyOriginalPath).(string)
	referer := r.Header.Get("Referer")

	subId := chi.URLParam(r, "subscriptionId")
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")

	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._preflightValidation(ctx, log, body, correlationData, systemData, r.URL.Path, originalPath, r.Method, referer, &header, f.apis[apiVersion].OpenShiftClusterConverter, f.apis[apiVersion].OpenShiftClusterStaticValidator, subId, resourceProviderNamespace, apiVersion)
		return err
	})

	frontendOperationResultLog(log, r.Method, err)
	reply(log, w, header, b, err)
}

func (f *frontend) _preflightValidation(ctx context.Context, log *logrus.Entry, body []byte, correlationData *api.CorrelationData, systemData *api.SystemData, path, originalPath, method, referer string, header *http.Header, converter api.OpenShiftClusterConverter, staticValidator api.OpenShiftClusterStaticValidator, subId, resourceProviderNamespace string, apiVersion string) ([]byte, error) {
	// multiple errors should be able to return, just add them to the message
	subscription, err := f.validateSubscriptionState(ctx, path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	doc, err := f.dbOpenShiftClusters.Get(ctx, path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	newProvisionState := doc == nil

	// dont create this as it would require creating openshiftclusterdoc
	// instead grab info you may need like location that is given to the doc
	/*
			originalR, err := azure.ParseResourceID(originalPath)
		if err != nil {
			return nil, err
		}

		doc = &api.OpenShiftClusterDocument{
			ID:  f.dbOpenShiftClusters.NewUUID(),
			Key: path,
			OpenShiftCluster: &api.OpenShiftCluster{
				ID:   originalPath,
				Name: originalR.ResourceName,
				Type: originalR.Provider + "/" + originalR.ResourceType,
				Properties: api.OpenShiftClusterProperties{
					ArchitectureVersion: version.InstallArchitectureVersion,
					ProvisioningState:   api.ProvisioningStateSucceeded,
					CreatedAt:           f.now().UTC(),
					CreatedBy:           version.GitCommit,
					ProvisionedBy:       version.GitCommit,
				},
			},
		}
		if !f.env.IsLocalDevelopmentMode() {
			doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled = true
		}
	*/

	if !newProvisionState {
		err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
		if err != nil {
			return nil, err
		}
	} else {
		// preflight doc mentions SKU and quota validation are good candidates to do preflight validation on
		err = f.skuValidator.ValidateVMSku(ctx, f.env.Environment(), f.env, subscription.ID, subscription.Subscription.Properties.TenantID, cluster)
		if err != nil {
			return nil, err
		}

		err = f.quotaValidator.ValidateQuota(ctx, f.env.Environment(), f.env, subscription.ID, subscription.Subscription.Properties.TenantID, cluster)
		if err != nil {
			return nil, err
		}

		// Not sure if we keep this one yet
		err = staticValidator.Static(ext, nil, f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sV3Workers), path)
		if err != nil {
			return nil, err
		}

		err = f.validateInstallVersion(ctx, doc)
		if err != nil {
			return nil, err
		}
	}

	return b, err
}
