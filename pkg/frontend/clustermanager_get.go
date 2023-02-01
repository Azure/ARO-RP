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
)

func (f *frontend) getClusterManagerConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	var (
		b   []byte
		err error
	)

	apiVersion, ocmResourceType := r.URL.Query().Get(api.APIVersionKey), vars["ocmResourceType"]

	err = f.validateOcmResourceType(apiVersion, ocmResourceType)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", err.Error())
		return
	}

	switch vars["ocmResourceType"] {
	case "syncset":
		b, err = f._getSyncSetConfiguration(ctx, log, r, f.apis[apiVersion].SyncSetConverter)
	case "machinepool":
		b, err = f._getMachinePoolConfiguration(ctx, log, r, f.apis[apiVersion].MachinePoolConverter)
	case "syncidentityprovider":
		b, err = f._getSyncIdentityProviderConfiguration(ctx, log, r, f.apis[apiVersion].SyncIdentityProviderConverter)
	case "secret":
		b, err = f._getSecretConfiguration(ctx, log, r, f.apis[apiVersion].SecretConverter)
	default:
		return
	}

	reply(log, w, nil, b, err)
}

func (f *frontend) _getSyncSetConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.SyncSetConverter) ([]byte, error) {
	vars := mux.Vars(r)

	resType, resName, ocmResType, ocmResName, resGroupName := vars["resourceType"], vars["resourceName"], vars["ocmResourceType"], vars["ocmResourceName"], vars["resourceGroupName"]
	doc, err := f.validateResourceForGet(ctx, resType, resName, ocmResType, ocmResName, resGroupName, r.URL.Path, r)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(doc.SyncSet)
	return json.MarshalIndent(ext, "", "    ")
}

func (f *frontend) _getMachinePoolConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.MachinePoolConverter) ([]byte, error) {
	vars := mux.Vars(r)

	resType, resName, ocmResType, ocmResName, resGroupName := vars["resourceType"], vars["resourceName"], vars["ocmResourceType"], vars["ocmResourceName"], vars["resourceGroupName"]
	doc, err := f.validateResourceForGet(ctx, resType, resName, ocmResType, ocmResName, resGroupName, r.URL.Path, r)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(doc.MachinePool)
	return json.MarshalIndent(ext, "", "    ")
}

func (f *frontend) _getSyncIdentityProviderConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.SyncIdentityProviderConverter) ([]byte, error) {
	vars := mux.Vars(r)

	resType, resName, ocmResType, ocmResName, resGroupName := vars["resourceType"], vars["resourceName"], vars["ocmResourceType"], vars["ocmResourceName"], vars["resourceGroupName"]
	doc, err := f.validateResourceForGet(ctx, resType, resName, ocmResType, ocmResName, resGroupName, r.URL.Path, r)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(doc.SyncIdentityProvider)
	return json.MarshalIndent(ext, "", "    ")
}

func (f *frontend) _getSecretConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.SecretConverter) ([]byte, error) {
	vars := mux.Vars(r)
	resType, resName, ocmResType, ocmResName, resGroupName := vars["resourceType"], vars["resourceName"], vars["ocmResourceType"], vars["ocmResourceName"], vars["resourceGroupName"]
	doc, err := f.validateResourceForGet(ctx, resType, resName, ocmResType, ocmResName, resGroupName, r.URL.Path, r)
	if err != nil {
		return nil, err
	}

	ext := converter.ToExternal(doc.Secret)
	return json.MarshalIndent(ext, "", "    ")
}

func (f *frontend) validateResourceForGet(ctx context.Context, resType, resName, ocmResType, ocmResName, resGroupName, path string, r *http.Request) (*api.ClusterManagerConfigurationDocument, error) {
	doc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	if err != nil {
		switch {
		case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
			return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' under resource group '%s' was not found.",
				resType, resName, ocmResType, ocmResName, resGroupName)
		default:
			return nil, err
		}
	}

	if doc.Deleting {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on a resource marked for deletion.")
	}

	return doc, nil
}
