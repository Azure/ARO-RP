package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) putAdminMaintManifestCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceID := resourceIdFromURLParams(r)
	b, err := f._putAdminMaintManifestCreate(ctx, r, resourceID, "")

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	err = statusCodeError(http.StatusCreated)
	adminReply(log, w, nil, b, err)
}

func (f *frontend) _putAdminMaintManifestCreate(ctx context.Context, r *http.Request, resourceID string, maintenanceTaskID string) ([]byte, error) {
	converter := f.apis[admin.APIVersion].MaintenanceManifestConverter
	validator := f.apis[admin.APIVersion].MaintenanceManifestStaticValidator

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

	var ext *admin.MaintenanceManifest

	if maintenanceTaskID != "" {
		ext = &admin.MaintenanceManifest{}
		ext.MaintenanceTaskID = maintenanceTaskID
	} else {
		body := r.Context().Value(middleware.ContextKeyBody).([]byte)
		if len(body) == 0 || !json.Valid(body) {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
		}
		err = json.Unmarshal(body, &ext)
		if err != nil {
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content could not be deserialized: "+err.Error())
		}
	}

	// fill in some defaults
	ext.ID = dbMaintenanceManifests.NewUUID()
	ext.State = admin.MaintenanceManifestStatePending

	if ext.RunAfter == 0 {
		ext.RunAfter = int(f.now().Unix())
	}

	// add a 7d timeout by default
	if ext.RunBefore == 0 {
		ext.RunBefore = int(f.now().Add(time.Hour * 7 * 24).Unix())
	}

	err = validator.Static(ext, nil)
	if err != nil {
		return nil, err
	}

	manifestDoc := &api.MaintenanceManifestDocument{
		ClusterResourceID: resourceID,
	}
	converter.ToInternal(ext, manifestDoc)

	savedDoc, err := dbMaintenanceManifests.Create(ctx, manifestDoc)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return json.MarshalIndent(converter.ToExternal(savedDoc, true), "", "    ")
}
