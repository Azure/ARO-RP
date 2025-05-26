package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

var validationSuccess = api.ValidationResult{
	Status: api.ValidationStatusSucceeded,
}

// Preflight always returns a 200 status
// /subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/deployments/{deploymentName}/preflight?api-version={api-version}
func (f *frontend) preflightValidation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	var header http.Header
	var b []byte

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)

	resources, err := unmarshalRequest(body)
	if err != nil {
		b = marshalValidationResult(api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.CloudErrorBody{
				Message: err.Error(),
			},
		})
		reply(log, w, header, b, statusCodeError(http.StatusOK))
	}

	for _, raw := range resources.Resources {
		// get typeMeta from the raw data
		typeMeta := api.ResourceTypeMeta{}
		if err := json.Unmarshal(raw, &typeMeta); err != nil {
			// failing to parse the preflight body is not considered a validation failure. continue
			log.Warningf("preflight validation failed with bad request. Failed to unmarshal ResourceTypeMeta: %s", err)
			b = marshalValidationResult(api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Message: err.Error(),
				},
			})
			reply(log, w, header, b, statusCodeError(http.StatusOK))
			return
		}
		if strings.EqualFold(typeMeta.Type, "Microsoft.RedHatOpenShift/openShiftClusters") {
			res := f._preflightValidation(ctx, log, raw, typeMeta.APIVersion, strings.ToLower(typeMeta.Id))
			if res.Status == api.ValidationStatusFailed {
				log.Warningf("preflight validation failed with err: '%s'", res.Error.Message)
				b = marshalValidationResult(res)
				reply(log, w, header, b, statusCodeError(http.StatusOK))
				return
			}
		}
	}

	log.Info("preflight validation succeeded")
	b = marshalValidationResult(validationSuccess)
	reply(log, w, header, b, statusCodeError(http.StatusOK))
}

func (f *frontend) _preflightValidation(ctx context.Context, log *logrus.Entry, raw json.RawMessage, apiVersion string, resourceID string) api.ValidationResult {
	log.Infof("running preflight validation on resource: %s", resourceID)
	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		log.Error(err)
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.CloudErrorBody{
				Message: fmt.Sprintf("500: %s", api.CloudErrorCodeInternalServerError),
			},
		}
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	isCreate := cosmosdb.IsErrorStatusCode(err, http.StatusNotFound)
	if err != nil && !isCreate {
		log.Warning(err.Error())
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.CloudErrorBody{
				Message: "400: Cluster not found for resourceID: " + resourceID,
			},
		}
	}
	// unmarshal raw to OpenShiftCluster type
	oc := &api.OpenShiftCluster{}
	oc.Properties.ProvisioningState = api.ProvisioningStateSucceeded

	if !f.env.IsLocalDevelopmentMode() /* not local dev or CI */ {
		oc.Properties.FeatureProfile.GatewayEnabled = true
	}

	converter := f.apis[apiVersion].OpenShiftClusterConverter
	staticValidator := f.apis[apiVersion].OpenShiftClusterStaticValidator
	ext := converter.ToExternal(oc)
	converter.ExternalNoReadOnly(ext)
	if err = json.Unmarshal(raw, &ext); err != nil {
		log.Warning(err.Error())
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.CloudErrorBody{
				Message: api.CloudErrorCodeInternalServerError,
			},
		}
	}
	if isCreate {
		converter.ToInternal(ext, oc)
		if err = staticValidator.Static(ext, nil, f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sWorkers), version.InstallArchitectureVersion, resourceID); err != nil {
			return api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Message: err.Error(),
				},
			}
		}
		if err := f.validateInstallVersion(ctx, oc); err != nil {
			return api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: err.Error(),
				},
			}
		}
	} else {
		if err := staticValidator.Static(ext, doc.OpenShiftCluster, f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sWorkers), version.InstallArchitectureVersion, resourceID); err != nil {
			return api.ValidationResult{
				Status: api.ValidationStatusFailed,
				Error: &api.CloudErrorBody{
					Message: err.Error(),
				},
			}
		}
	}
	converter.ToInternal(ext, oc)
	if err := f.validatePlatformWorkloadIdentities(oc); err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeInvalidParameter,
				Message: err.Error(),
			},
		}
	}

	return validationSuccess
}

func unmarshalRequest(body []byte) (*api.PreflightRequest, error) {
	preflightRequest := &api.PreflightRequest{}
	if err := json.Unmarshal(body, preflightRequest); err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", fmt.Sprintf("The request content was invalid and could not be deserialized: %q.", err))
	}
	return preflightRequest, nil
}

func marshalValidationResult(results api.ValidationResult) []byte {
	body, err := json.Marshal(results)
	if err != nil {
		return nil
	} else {
		return body
	}
}
