package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) putAdminMaintScheduleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	b, err := f._putAdminMaintScheduleCreate(ctx, r)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	err = statusCodeError(http.StatusCreated)
	adminReply(log, w, nil, b, err)
}

func (f *frontend) _putAdminMaintScheduleCreate(ctx context.Context, r *http.Request) ([]byte, error) {
	converter := f.apis[admin.APIVersion].MaintenanceScheduleConverter
	validator := f.apis[admin.APIVersion].MaintenanceScheduleStaticValidator

	dbMaintenanceSchedules, err := f.dbGroup.MaintenanceSchedules()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	var ext *admin.MaintenanceSchedule

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) == 0 || !json.Valid(body) {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
	}
	err = json.Unmarshal(body, &ext)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content could not be deserialized: "+err.Error())
	}

	// fill in the ID
	ext.ID = dbMaintenanceSchedules.NewUUID()

	err = validator.Static(ext, nil)
	if err != nil {
		return nil, err
	}

	scheduleDoc := &api.MaintenanceScheduleDocument{}
	converter.ToInternal(ext, scheduleDoc)

	savedDoc, err := dbMaintenanceSchedules.Create(ctx, scheduleDoc)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return json.MarshalIndent(converter.ToExternal(savedDoc), "", "    ")
}
