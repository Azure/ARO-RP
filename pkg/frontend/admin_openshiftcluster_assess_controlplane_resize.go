package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

type DownsizeResponse struct {
	Proceed        bool   `json:"proceed"`
	Recommendation string `json:"recommendation"`
	Details        string `json:"details"`
}

func (f *frontend) postAdminAssessControlPlaneDownsize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	b, err := f._postAdminAssessControlPlaneDownsize(ctx, r, log)
	adminReply(log, w, nil, b, err)
}

func (f *frontend) _postAdminAssessControlPlaneDownsize(
	ctx context.Context,
	r *http.Request,
	log *logrus.Entry,
) ([]byte, error) {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	resourceID = strings.TrimSuffix(resourceID, "/assesscontrolplanedownsize")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)

	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(
			http.StatusNotFound, api.CloudErrorCodeResourceNotFound,
			"",
			fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName),
		)
	case err != nil:
		return nil, err
	}

	portFwdAction, err := f.portForwardActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	cpResizeMgr := newControlPlaneResize(portFwdAction)
	assessment, err := cpResizeMgr.assessDownsizeRequest(ctx, log)

	assessmentJsonStr, err := json.MarshalIndent(assessment, "", " ")
	if err != nil {
		log.Errorf("Error marshaling assessment: %v", err)
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	log.Infof("Downsize assessment: %s", assessmentJsonStr)

	return assessmentJsonStr, nil

}
