package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
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
	resourceGroupName := chi.URLParam(r, "resourceGroupName")

	resources, err := unmarshalRequest(body)
	if err != nil {
		log.Warningf("Bad Request. The request content was invalid and could not be deserialized: %s.", err)
	} else {
		resourceCount := len(resources.Resources)
		ch := make(chan api.ValidationResult, resourceCount)
		for _, raw := range resources.Resources {
			// get typeMeta from the raw data
			typeMeta := &api.ResourceTypeMeta{}
			err = json.Unmarshal(raw, &typeMeta)
			if err != nil {
				// failing to parse the preflight body is not considered a validation failure. continue
				log.Warningf("Bad request. Failed to unmarshal ResourceTypeMeta: %s", err)
				continue
			}
			if strings.EqualFold(typeMeta.Type, "Microsoft.RedHatOpenShift/openShiftClusters") {
				path := PreflightResourceId(subId, resourceGroupName, typeMeta.Name)
				ch <- f._preflightValidation(ctx, log, raw, correlationData, systemData, path, originalPath, r.Method, referer, &header, f.apis[typeMeta.APIVersion].OpenShiftClusterConverter, f.apis[typeMeta.APIVersion].OpenShiftClusterStaticValidator, subId, typeMeta.APIVersion, resourceGroupName)
			}
		}
		close(ch)

		for res := range ch {
			// go through and serialize into an array and make that the return body
			if res.Status == api.ValidationStatusFailed {
				log.Error("preflight validation failed with: %s", res.Error)
				resultByte, err := json.Marshal(res.Error)
				if err != nil {
					log.Warningf("The response could not be serialized: %s.", err)
				} else {
					b = append(b, resultByte...)
				}
			}
		}
	}
	reply(log, w, header, b, nil)
}

func (f *frontend) _preflightValidation(ctx context.Context, log *logrus.Entry, raw json.RawMessage, correlationData *api.CorrelationData, systemData *api.SystemData, path, originalPath, method, referer string, header *http.Header, converter api.OpenShiftClusterConverter, staticValidator api.OpenShiftClusterStaticValidator, subId, apiVersion string, resourceGroupName string) api.ValidationResult {
	subscription, err := f.validateSubscriptionState(ctx, path, api.SubscriptionStateRegistered)
	if err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error:  err,
		}
	}
	// portal extentsion repo should have a template
	// unmarshal raw to OpenShiftCluster type
	doc := &api.OpenShiftCluster{}
	if !f.env.IsLocalDevelopmentMode() /* not local dev or CI */ {
		doc.Properties.FeatureProfile.GatewayEnabled = true
	}

	var ext interface{}
	ext = converter.ToExternal(doc)
	err = json.Unmarshal(raw, &ext)
	if err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error:  err,
		}
	}

	// For Put operation
	converter.ToInternal(ext, doc)
	err = staticValidator.Static(ext, nil, f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sV3Workers), path)
	if err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error:  err,
		}
	}

	// NOTE: Check if doc is set here or blank
	err = f.skuValidator.ValidateVMSku(ctx, f.env.Environment(), f.env, subscription.ID, subscription.Subscription.Properties.TenantID, doc.Location, string(doc.Properties.MasterProfile.VMSize), doc.Properties.WorkerProfiles)
	if err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error:  err,
		}
	}

	err = f.validateInstallVersion(ctx, doc)
	if err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error:  err,
		}
	}

	return validationSuccess
}

func unmarshalRequest(body []byte) (*api.PreflightRequest, error) {
	preflightRequest := &api.PreflightRequest{}
	if err := json.Unmarshal(body, preflightRequest); err != nil {
		return nil, fmt.Errorf("Failed to ummarshal preflightRequest: %w", err)
	}
	return preflightRequest, nil
}

// String function returns a string in form of azureResourceID
func PreflightResourceId(subId string, resourceGroupName string, resourceName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subId, resourceGroupName, resourceName)
}

var validationSuccess = api.ValidationResult{
	Status: api.ValidationStatusSucceeded,
}
