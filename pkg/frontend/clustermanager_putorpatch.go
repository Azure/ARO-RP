package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

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

	var (
		header http.Header
		b      []byte
		err    error
	)

	apiVersion, ocmResourceType := r.URL.Query().Get(api.APIVersionKey), vars["ocmResourceType"]

	err = f.validateOcmResourceType(apiVersion, ocmResourceType)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", err.Error())
		return
	}

	err = cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		switch ocmResourceType {
		case "syncset":
			b, err = f._putOrPatchSyncSet(ctx, log, r, &header, f.apis[apiVersion].SyncSetConverter, f.apis[apiVersion].ClusterManagerStaticValidator)
		case "machinepool":
			b, err = f._putOrPatchMachinePool(ctx, log, r, &header, f.apis[apiVersion].MachinePoolConverter, f.apis[apiVersion].ClusterManagerStaticValidator)
		case "syncidentityprovider":
			b, err = f._putOrPatchSyncIdentityProvider(ctx, log, r, &header, f.apis[apiVersion].SyncIdentityProviderConverter, f.apis[apiVersion].ClusterManagerStaticValidator)
		case "secret":
			b, err = f._putOrPatchSecret(ctx, log, r, &header, f.apis[apiVersion].SecretConverter, f.apis[apiVersion].ClusterManagerStaticValidator)
		}
		return err
	})

	reply(log, w, header, b, err)
}

func (f *frontend) _putOrPatchSyncSet(ctx context.Context, log *logrus.Entry, r *http.Request, header *http.Header, converter api.SyncSetConverter, staticValidator api.ClusterManagerStaticValidator) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]
	ocmResourceType, ocmResourceName := vars["ocmResourceType"], vars["ocmResourceName"]
	originalPath, err := f.extractOriginalPath(ctx, r, resType, resName, resGroupName)
	if err != nil {
		return nil, err
	}

	ocmdoc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	} else if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) && r.Method == http.MethodPatch {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' under resource group '%s' was not found.",
			resType, resName, ocmResourceType, ocmResourceName, resGroupName)
	}

	var resources string
	err = json.Unmarshal(body, &resources)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	err = staticValidator.Static(resources, vars)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The 'Kind' in the request payload does not match 'Kind' in the request path: %q.", err)
	}

	isCreate := ocmdoc == nil
	uuid := f.dbClusterManagerConfiguration.NewUUID()
	if isCreate {
		ocmdoc = &api.ClusterManagerConfigurationDocument{
			ID:  uuid,
			Key: r.URL.Path,
		}
		ocmdoc.SyncSet = &api.SyncSet{
			Name: ocmResourceName,
			Type: "Microsoft.RedHatOpenShift/SyncSet",
			ID:   originalPath,
			Properties: api.SyncSetProperties{
				Resources: resources,
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
		ocmdoc.SyncSet.Properties.Resources = resources
	}

	ocmdoc.CorrelationData = correlationData
	f.systemDataSyncSetEnricher(ocmdoc, systemData)

	ocmdoc, err = f.dbClusterManagerConfiguration.Update(ctx, ocmdoc)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(ocmdoc.SyncSet)
	b, err := json.MarshalIndent(ext, "", "  ")
	return b, err
}

func (f *frontend) _putOrPatchMachinePool(ctx context.Context, log *logrus.Entry, r *http.Request, header *http.Header, converter api.MachinePoolConverter, staticValidator api.ClusterManagerStaticValidator) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]
	ocmResourceType, ocmResourceName := vars["ocmResourceType"], vars["ocmResourceName"]

	originalPath, err := f.extractOriginalPath(ctx, r, resType, resName, resGroupName)
	if err != nil {
		return nil, err
	}

	ocmdoc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	} else if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) && r.Method == http.MethodPatch {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' under resource group '%s' was not found.",
			resType, resName, ocmResourceType, ocmResourceName, resGroupName)
	}

	var resources string
	err = json.Unmarshal(body, &resources)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	err = staticValidator.Static(resources, vars)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The 'Kind' in the request payload does not match 'Kind' in the request path: %q.", err)
	}

	isCreate := ocmdoc == nil
	uuid := f.dbClusterManagerConfiguration.NewUUID()
	if isCreate {
		ocmdoc = &api.ClusterManagerConfigurationDocument{
			ID:  uuid,
			Key: r.URL.Path,
		}
		ocmdoc.MachinePool = &api.MachinePool{
			Name: ocmResourceName,
			Type: "Microsoft.RedHatOpenShift/MachinePool",
			ID:   originalPath,
			Properties: api.MachinePoolProperties{
				Resources: resources,
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
		ocmdoc.MachinePool.Properties.Resources = resources
	}

	ocmdoc.CorrelationData = correlationData
	f.systemDataMachinePoolEnricher(ocmdoc, systemData)

	ocmdoc, err = f.dbClusterManagerConfiguration.Update(ctx, ocmdoc)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(ocmdoc.MachinePool)
	b, err := json.MarshalIndent(ext, "", "  ")
	return b, err
}

func (f *frontend) _putOrPatchSyncIdentityProvider(ctx context.Context, log *logrus.Entry, r *http.Request, header *http.Header, converter api.SyncIdentityProviderConverter, staticValidator api.ClusterManagerStaticValidator) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]
	ocmResourceType, ocmResourceName := vars["ocmResourceType"], vars["ocmResourceName"]

	originalPath, err := f.extractOriginalPath(ctx, r, resType, resName, resGroupName)
	if err != nil {
		return nil, err
	}

	ocmdoc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	} else if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) && r.Method == http.MethodPatch {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' under resource group '%s' was not found.",
			resType, resName, ocmResourceType, ocmResourceName, resGroupName)
	}

	var resources string
	err = json.Unmarshal(body, &resources)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	err = staticValidator.Static(resources, vars)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The 'Kind' in the request payload does not match 'Kind' in the request path: %q.", err)
	}

	isCreate := ocmdoc == nil
	uuid := f.dbClusterManagerConfiguration.NewUUID()
	if isCreate {
		ocmdoc = &api.ClusterManagerConfigurationDocument{
			ID:  uuid,
			Key: r.URL.Path,
		}
		ocmdoc.SyncIdentityProvider = &api.SyncIdentityProvider{
			Name: ocmResourceName,
			Type: "Microsoft.RedHatOpenShift/SyncIdentityProvider",
			ID:   originalPath,
			Properties: api.SyncIdentityProviderProperties{
				Resources: resources,
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
		ocmdoc.SyncIdentityProvider.Properties.Resources = resources
	}

	ocmdoc.CorrelationData = correlationData
	f.systemDataSyncIdentityProviderEnricher(ocmdoc, systemData)

	ocmdoc, err = f.dbClusterManagerConfiguration.Update(ctx, ocmdoc)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(ocmdoc.SyncIdentityProvider)
	b, err := json.MarshalIndent(ext, "", "  ")
	return b, err
}

func (f *frontend) _putOrPatchSecret(ctx context.Context, log *logrus.Entry, r *http.Request, header *http.Header, converter api.SecretConverter, staticValidator api.ClusterManagerStaticValidator) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]
	ocmResourceType, ocmResourceName := vars["ocmResourceType"], vars["ocmResourceName"]

	originalPath, err := f.extractOriginalPath(ctx, r, resType, resName, resGroupName)
	if err != nil {
		return nil, err
	}

	ocmdoc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	} else if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) && r.Method == http.MethodPatch {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' under resource group '%s' was not found.",
			resType, resName, ocmResourceType, ocmResourceName, resGroupName)
	}

	var resources string
	err = json.Unmarshal(body, &resources)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	err = staticValidator.Static(resources, vars)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The 'Kind' in the request payload does not match 'Kind' in the request path: %q.", err)
	}

	isCreate := ocmdoc == nil
	uuid := f.dbClusterManagerConfiguration.NewUUID()
	if isCreate {
		ocmdoc = &api.ClusterManagerConfigurationDocument{
			ID:  uuid,
			Key: r.URL.Path,
		}
		ocmdoc.Secret = &api.Secret{
			Name: ocmResourceName,
			Type: "Microsoft.RedHatOpenShift/Secret",
			ID:   originalPath,
			Properties: api.SecretProperties{
				SecretResources: api.SecureString(resources),
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
		ocmdoc.Secret.Properties.SecretResources = api.SecureString(resources)
	}

	ocmdoc.CorrelationData = correlationData
	f.systemDataSecretEnricher(ocmdoc, systemData)

	ocmdoc, err = f.dbClusterManagerConfiguration.Update(ctx, ocmdoc)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(ocmdoc.Secret)
	b, err := json.MarshalIndent(ext, "", "  ")
	return b, err
}

func (f *frontend) extractOriginalPath(ctx context.Context, r *http.Request, resType, resName, resGroupName string) (string, error) {
	_, err := f.validateSubscriptionState(ctx, r.URL.Path, api.SubscriptionStateRegistered)
	if err != nil {
		return "", err
	}

	originalPath := r.Context().Value(middleware.ContextKeyOriginalPath).(string)
	armResource, err := arm.ParseArmResourceId(originalPath)
	if err != nil {
		return "", err
	}

	ocp, err := f.dbOpenShiftClusters.Get(ctx, armResource.ParentResource())
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return "", err
	}

	if ocp == nil || cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return "", api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	}

	return originalPath, err
}

// TODO once we hit go1.18 we can refactor to use generics for any document using systemData
// enrichClusterManagerSystemData will selectively overwrite systemData fields based on
// arm inputs
func enrichSyncSetSystemData(doc *api.ClusterManagerConfigurationDocument, systemData *api.SystemData) {
	if systemData == nil {
		return
	}
	if doc.SyncSet.SystemData == nil {
		doc.SyncSet.SystemData = &api.SystemData{}
	}
	if systemData.CreatedAt != nil {
		doc.SyncSet.SystemData.CreatedAt = systemData.CreatedAt
	}
	if systemData.CreatedBy != "" {
		doc.SyncSet.SystemData.CreatedBy = systemData.CreatedBy
	}
	if systemData.CreatedByType != "" {
		doc.SyncSet.SystemData.CreatedByType = systemData.CreatedByType
	}
	if systemData.LastModifiedAt != nil {
		doc.SyncSet.SystemData.LastModifiedAt = systemData.LastModifiedAt
	}
	if systemData.LastModifiedBy != "" {
		doc.SyncSet.SystemData.LastModifiedBy = systemData.LastModifiedBy
	}
	if systemData.LastModifiedByType != "" {
		doc.SyncSet.SystemData.LastModifiedByType = systemData.LastModifiedByType
	}
}

func enrichMachinePoolSystemData(doc *api.ClusterManagerConfigurationDocument, systemData *api.SystemData) {
	if systemData == nil {
		return
	}
	if doc.MachinePool.SystemData == nil {
		doc.MachinePool.SystemData = &api.SystemData{}
	}
	if systemData.CreatedAt != nil {
		doc.MachinePool.SystemData.CreatedAt = systemData.CreatedAt
	}
	if systemData.CreatedBy != "" {
		doc.MachinePool.SystemData.CreatedBy = systemData.CreatedBy
	}
	if systemData.CreatedByType != "" {
		doc.MachinePool.SystemData.CreatedByType = systemData.CreatedByType
	}
	if systemData.LastModifiedAt != nil {
		doc.MachinePool.SystemData.LastModifiedAt = systemData.LastModifiedAt
	}
	if systemData.LastModifiedBy != "" {
		doc.MachinePool.SystemData.LastModifiedBy = systemData.LastModifiedBy
	}
	if systemData.LastModifiedByType != "" {
		doc.MachinePool.SystemData.LastModifiedByType = systemData.LastModifiedByType
	}
}

func enrichSyncIdentityProviderSystemData(doc *api.ClusterManagerConfigurationDocument, systemData *api.SystemData) {
	if systemData == nil {
		return
	}
	if doc.SyncIdentityProvider.SystemData == nil {
		doc.SyncIdentityProvider.SystemData = &api.SystemData{}
	}
	if systemData.CreatedAt != nil {
		doc.SyncIdentityProvider.SystemData.CreatedAt = systemData.CreatedAt
	}
	if systemData.CreatedBy != "" {
		doc.SyncIdentityProvider.SystemData.CreatedBy = systemData.CreatedBy
	}
	if systemData.CreatedByType != "" {
		doc.SyncIdentityProvider.SystemData.CreatedByType = systemData.CreatedByType
	}
	if systemData.LastModifiedAt != nil {
		doc.SyncIdentityProvider.SystemData.LastModifiedAt = systemData.LastModifiedAt
	}
	if systemData.LastModifiedBy != "" {
		doc.SyncIdentityProvider.SystemData.LastModifiedBy = systemData.LastModifiedBy
	}
	if systemData.LastModifiedByType != "" {
		doc.SyncIdentityProvider.SystemData.LastModifiedByType = systemData.LastModifiedByType
	}
}

func enrichSecretSystemData(doc *api.ClusterManagerConfigurationDocument, systemData *api.SystemData) {
	if systemData == nil {
		return
	}
	if doc.Secret.SystemData == nil {
		doc.Secret.SystemData = &api.SystemData{}
	}
	if systemData.CreatedAt != nil {
		doc.Secret.SystemData.CreatedAt = systemData.CreatedAt
	}
	if systemData.CreatedBy != "" {
		doc.Secret.SystemData.CreatedBy = systemData.CreatedBy
	}
	if systemData.CreatedByType != "" {
		doc.Secret.SystemData.CreatedByType = systemData.CreatedByType
	}
	if systemData.LastModifiedAt != nil {
		doc.Secret.SystemData.LastModifiedAt = systemData.LastModifiedAt
	}
	if systemData.LastModifiedBy != "" {
		doc.Secret.SystemData.LastModifiedBy = systemData.LastModifiedBy
	}
	if systemData.LastModifiedByType != "" {
		doc.Secret.SystemData.LastModifiedByType = systemData.LastModifiedByType
	}
}
