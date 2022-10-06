package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func (f *frontend) putOrPatchClusterManagerConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)
	var header http.Header
	var b []byte

	if f.apis[vars["api-version"]].ClusterManagerConfigurationConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], vars["api-version"])
		return
	}

	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putOrPatchClusterManagerConfiguration(ctx, log, r, &header, f.apis[vars["api-version"]].ClusterManagerConfigurationConverter)
		return err
	})

	reply(log, w, header, b, err)
}

func (f *frontend) _putOrPatchClusterManagerConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, header *http.Header, converter api.ClusterManagerConfigurationConverter) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	vars := mux.Vars(r)

	_, err := f.validateSubscriptionState(ctx, r.URL.Path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	originalPath := r.Context().Value(middleware.ContextKeyOriginalPath).(string)
	armResource, err := arm.ParseArmResourceId(originalPath)
	if err != nil {
		return nil, err
	}

	ocp, err := f.dbOpenShiftClusters.Get(ctx, armResource.ParentResource())
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	if ocp == nil || cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	}

	ocmdoc, _ := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	var resources string
	err = json.Unmarshal(body, &resources)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	isCreate := ocmdoc == nil
	uuid := f.dbClusterManagerConfiguration.NewUUID()
	if isCreate {
		ocmdoc = &api.ClusterManagerConfigurationDocument{
			ID:  uuid,
			Key: r.URL.Path,
			ClusterManagerConfiguration: &api.ClusterManagerConfiguration{
				ID:                originalPath,
				Name:              armResource.SubResource.ResourceName,
				ClusterResourceID: strings.ToLower(armResource.ParentResource()),
				Properties: api.ClusterManagerConfigurationProperties{
					Resources: []byte(resources),
				},
			},
		}

		var newdoc *api.ClusterManagerConfigurationDocument
		err = cosmosdb.RetryOnPreconditionFailed(func() error {
			newdoc, err = f.dbClusterManagerConfiguration.Create(ctx, ocmdoc)
			return err
		})
		ocmdoc = newdoc
	} else {
		if ocmdoc.Deleting {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on a resource marked for deletion.")
		}
		if ocmdoc.ClusterManagerConfiguration != nil {
			ocmdoc.ClusterManagerConfiguration.Properties.Resources = []byte(resources)
		}
	}

	ocmdoc.CorrelationData = correlationData

	f.systemDataClusterManagerEnricher(ocmdoc, systemData)
	ocmdoc, err = f.dbClusterManagerConfiguration.Update(ctx, ocmdoc)
	if err != nil {
		return nil, err
	}

	var ext interface{}
	ext, err = converter.ToExternal(ocmdoc.ClusterManagerConfiguration)
	if err != nil {
		return nil, err
	}

	b, err := json.MarshalIndent(ext, "", "  ")
	return b, err
}

// TODO once we hit go1.18 we can refactor to use generics for any document using systemData
// enrichClusterManagerSystemData will selectively overwrite systemData fields based on
// arm inputs
func enrichClusterManagerSystemData(doc *api.ClusterManagerConfigurationDocument, systemData *api.SystemData) {
	if systemData == nil {
		return
	}
	if systemData.CreatedAt != nil {
		doc.ClusterManagerConfiguration.SystemData.CreatedAt = systemData.CreatedAt
	}
	if systemData.CreatedBy != "" {
		doc.ClusterManagerConfiguration.SystemData.CreatedBy = systemData.CreatedBy
	}
	if systemData.CreatedByType != "" {
		doc.ClusterManagerConfiguration.SystemData.CreatedByType = systemData.CreatedByType
	}
	if systemData.LastModifiedAt != nil {
		doc.ClusterManagerConfiguration.SystemData.LastModifiedAt = systemData.LastModifiedAt
	}
	if systemData.LastModifiedBy != "" {
		doc.ClusterManagerConfiguration.SystemData.LastModifiedBy = systemData.LastModifiedBy
	}
	if systemData.LastModifiedByType != "" {
		doc.ClusterManagerConfiguration.SystemData.LastModifiedByType = systemData.LastModifiedByType
	}
}
