package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/go-autorest/autorest/azure"
)

// NOTE: Make sure to change and always return 200, with any errors inside of the response message
// /subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/deployments/{deploymentName}/preflight?api-version={api-version}
func (f *frontend) preflightValidation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	var header http.Header
	var b []byte

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData)
	originalPath := r.Context().Value(middleware.ContextKeyOriginalPath).(string)
	referer := r.Header.Get("Referer")

	subId := chi.URLParam(r, "subscriptionId")
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")

	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	b, err := f._preflightValidation(ctx, log, body, correlationData, systemData, r.URL.Path, originalPath, r.Method, referer, &header, f.apis[apiVersion].OpenShiftClusterConverter, f.apis[apiVersion].OpenShiftClusterStaticValidator, subId, resourceProviderNamespace, apiVersion)

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

	var ext interface{}

	if !newProvisionState {
		err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
		if err != nil {
			return nil, err
		}

		ext = converter.ToExternal(doc.OpenShiftCluster)
		err = json.Unmarshal(body, &ext)
		if err != nil {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
		}
		err = staticValidator.Static(ext, doc.OpenShiftCluster, f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sV3Workers), path)
		if err != nil {
			return nil, err
		}

	} else {
		originalR, err := azure.ParseResourceID(originalPath)
		if err != nil {
			return nil, err
		}

		ext = converter.ToExternal(&api.OpenShiftCluster{
			ID:   originalPath,
			Name: originalR.ResourceName,
			Type: originalR.Provider + "/" + originalR.ResourceType,
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: api.ProvisioningStateSucceeded,
			},
		})

		converter.ToInternal(ext, doc.OpenShiftCluster) // other idea would be to parse body with custom unmarshal

		// Validate static on payload

		err = f.skuValidator.ValidateVMSku(ctx, f.env.Environment(), f.env, subscription.ID, subscription.Subscription.Properties.TenantID, doc.OpenShiftCluster.Location, string(doc.OpenShiftCluster.Properties.MasterProfile.VMSize), doc.OpenShiftCluster.Properties.WorkerProfiles)
		if err != nil {
			return nil, err
		}

		err = f.validateInstallVersion(ctx, doc)
		if err != nil {
			return nil, err
		}
	}

	return nil, err
}
