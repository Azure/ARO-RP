package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) putOrPatchClusterManagerConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	var header http.Header
	var b []byte
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putOrPatchClusterManagerConfiguration(ctx, log, r, &header, f.apis[vars["api-version"]].ClusterManagerConverter())
		return err
	})

	reply(log, w, header, b, err)
}

func (f *frontend) _putOrPatchClusterManagerConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, header *http.Header, converter api.ClusterManagerConverter) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	vars := mux.Vars(r)

	f.baseLog.Info("body: ", string(body))
	f.baseLog.Info("correlationData: ", correlationData)
	f.baseLog.Info("systemData: ", systemData)

	_, err := f.validateSubscriptionState(ctx, r.URL.Path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	originalPath := r.Context().Value(middleware.ContextKeyOriginalPath).(string)

	// this func isn't meant for sub resources, its going to take the name of our ocm resource
	// and use that for the cluster resourceID
	// until I fix this I just create ocm resources with the same name as the cluster :-)
	// TODO look into existing funcs vs a new splitting func
	cluster, err := azure.ParseResourceID(originalPath)
	if err != nil {
		return nil, err
	}
	clusterURL := strings.ToLower(cluster.String())

	ocpdoc, err := f.dbOpenShiftClusters.Get(ctx, clusterURL)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	exists := ocpdoc != nil
	if !exists {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "Cannot modify cluster resources when cluster does not exist.")
	}

	ocmdoc, _ := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	isCreate := ocmdoc == nil
	uuid := f.dbClusterManagerConfiguration.NewUUID()
	f.baseLog.Info("uuid: ", uuid)
	if isCreate {
		ocmdoc = &api.ClusterManagerConfigurationDocument{
			ID:  uuid,
			Key: r.URL.Path,
			ClusterManagerConfiguration: &api.ClusterManagerConfiguration{
				ID:                originalPath,
				ClusterResourceId: clusterURL,
				Resources:         body,
			},
		}
		switch vars["clusterManagerKind"] {
		case strings.ToLower(api.MachinePoolType):
			ocmdoc.ClusterManagerConfiguration.Kind = api.MachinePoolKind
		case strings.ToLower(api.SyncIdentityProviderType):
			ocmdoc.ClusterManagerConfiguration.Kind = api.SyncIdentityProviderKind
		case strings.ToLower(api.SyncSetType):
			ocmdoc.ClusterManagerConfiguration.Kind = api.SyncSetKind
		default:
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidResource, "", "Invalid cluster manager kind.")
		}
		ocmdoc.CorrelationData = correlationData

		newdoc, err := f.dbClusterManagerConfiguration.Create(ctx, ocmdoc)
		if err != nil {
			return nil, err
		}
		ocmdoc = newdoc
	}

	var ext interface{}
	ext, err = converter.ToExternal(ocmdoc.ClusterManagerConfiguration)
	if err != nil {
		f.baseLog.Fatal(err)
	}
	b, err := json.MarshalIndent(ext, "", "  ")
	return b, err
}
