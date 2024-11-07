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
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getSingleAdminMaintManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceID := resourceIdFromURLParams(r)
	b, err := f._getSingleAdminMaintManifest(ctx, r, resourceID)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getSingleAdminMaintManifest(ctx context.Context, r *http.Request, resourceID string) ([]byte, error) {
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

	manifest, err := dbMaintenanceManifests.Get(ctx, resourceID, manifestId)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", fmt.Sprintf("manifest not found: %s", err.Error()))
	}

	return json.MarshalIndent(converter.ToExternal(manifest, true), "", "    ")
}
