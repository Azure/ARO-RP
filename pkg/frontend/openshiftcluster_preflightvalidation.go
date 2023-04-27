package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
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
	if err == nil {
		for _, raw := range resources.Resources {
			// get typeMeta from the raw data
			typeMeta := api.ResourceTypeMeta{}
			if err := json.Unmarshal(raw, &typeMeta); err != nil {
				// failing to parse the preflight body is not considered a validation failure. continue
				log.Warningf("bad request. Failed to unmarshal ResourceTypeMeta: %s", err)
				continue
			}
			if strings.EqualFold(typeMeta.Type, "Microsoft.RedHatOpenShift/openShiftClusters") {
				res := f._preflightValidation(ctx, log, raw, typeMeta.APIVersion)
				if res.Status == api.ValidationStatusFailed {
					log.Warningf("preflight validation failed")
					b = marshalValidationResult(res)
					reply(log, w, header, b, statusCodeError(http.StatusOK))
					return
				}
			}
		}
		log.Info("preflight validation succeeded")
		b = marshalValidationResult(validationSuccess)
		reply(log, w, header, b, statusCodeError(http.StatusOK))
	} else {
		b = marshalValidationResult(api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.ManagementErrorWithDetails{
				Message: to.StringPtr(err.Error()),
			},
		})
		reply(log, w, header, b, statusCodeError(http.StatusOK))
	}
}

func (f *frontend) _preflightValidation(ctx context.Context, log *logrus.Entry, raw json.RawMessage, apiVersion string) api.ValidationResult {
	// unmarshal raw to OpenShiftCluster type
	doc := &api.OpenShiftCluster{}
	doc.Properties.ProvisioningState = api.ProvisioningStateSucceeded

	if !f.env.IsLocalDevelopmentMode() /* not local dev or CI */ {
		doc.Properties.FeatureProfile.GatewayEnabled = true
	}

	converter := f.apis[apiVersion].OpenShiftClusterConverter
	ext := converter.ToExternal(doc)
	if err := json.Unmarshal(raw, &ext); err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.ManagementErrorWithDetails{
				Message: to.StringPtr(err.Error()),
			},
		}
	}

	converter.ToInternal(ext, doc)

	if err := f.validateInstallVersion(ctx, doc); err != nil {
		return api.ValidationResult{
			Status: api.ValidationStatusFailed,
			Error: &api.ManagementErrorWithDetails{
				Code:    to.StringPtr(api.CloudErrorCodeInvalidParameter),
				Message: to.StringPtr(err.Error()),
			},
		}
	}

	return validationSuccess
}

func unmarshalRequest(body []byte) (*api.PreflightRequest, error) {
	preflightRequest := &api.PreflightRequest{}
	if err := json.Unmarshal(body, preflightRequest); err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
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
