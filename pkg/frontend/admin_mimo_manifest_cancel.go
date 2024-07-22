package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminMaintManifestCancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceID := resourceIdFromURLParams(r)
	b, err := f._postAdminMaintManifestCancel(ctx, r, resourceID)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _postAdminMaintManifestCancel(ctx context.Context, r *http.Request, resourceID string) ([]byte, error) {
	manifestId := chi.URLParam(r, "manifestId")

	converter := f.apis[admin.APIVersion].MaintenanceManifestConverter

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	dbMaintenanceManifests, err := f.dbGroup.MaintenanceManifests()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", fmt.Sprintf("cluster not found: %s", err.Error()))
	}

	if doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateDeleting {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "cluster being deleted")
	}

	modifiedDoc, err := dbMaintenanceManifests.Patch(ctx, resourceID, manifestId, func(mmd *api.MaintenanceManifestDocument) error {
		if mmd.MaintenanceManifest.State != api.MaintenanceManifestStatePending {
			return api.NewCloudError(http.StatusNotAcceptable, api.CloudErrorCodePropertyChangeNotAllowed, "", fmt.Sprintf("cannot cancel task in state %s", mmd.MaintenanceManifest.State))
		}

		mmd.MaintenanceManifest.State = api.MaintenanceManifestStateCancelled
		return nil
	})
	if err != nil {
		cloudErr, ok := err.(*api.CloudError)
		if ok {
			return nil, cloudErr
		} else if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
			return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", fmt.Sprintf("manifest not found: %s", err.Error()))
		} else {
			return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
		}
	}

	return json.MarshalIndent(converter.ToExternal(modifiedDoc), "", "    ")
}
