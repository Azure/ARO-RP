package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

type downsizeRequestParams struct {
	currentInstanceMemorySize int64
	currentInstanceCPUSize    int64
	targetInstanceMemorySize  int64
	targetInstanceCPUSize     int64
	numControlPlaneNodes      int64
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

	downsizeReq, err := newDownsizeRequestParams(r, log)
	if err != nil {
		return nil, err
	}

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
	assessment, err := cpResizeMgr.assessDownsizeRequest(ctx, downsizeReq, log)

	assessmentJsonStr, err := json.MarshalIndent(assessment, "", " ")
	if err != nil {
		log.Errorf("Error marshaling assessment: %v", err)
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	log.Infof("Downsize assessment: %s", assessmentJsonStr)

	return assessmentJsonStr, nil

}

func newDownsizeRequestParams(r *http.Request, log *logrus.Entry) (*downsizeRequestParams, error) {

	currentInstanceMemorySize, err := strconv.ParseInt(r.URL.Query().Get("instanceMemSize"), 10, 64)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("current instance memory size parameter is not an integer"),
		)
	}

	currentInstanceCPUSize, err := strconv.ParseInt(r.URL.Query().Get("instanceCPUSize"), 10, 64)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("current instance cpu size parameter is not an integer"),
		)
	}

	targetInstanceMemorySize, err := strconv.ParseInt(r.URL.Query().Get("targetInstanceMemSize"), 10, 64)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("target instance memory size parameter is not an integer"),
		)
	}

	targetInstanceCPUSize, err := strconv.ParseInt(r.URL.Query().Get("targetInstanceCPUSize"), 10, 64)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("target instance cpu size parameter is not an integer"),
		)
	}

	numControlPlaneNodes, err := strconv.ParseInt(r.URL.Query().Get("numControlPlaneNodes"), 10, 64)
	if err != nil {
		return nil, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("number of control plane nodes parameter is not an integer"),
		)
	}

	if currentInstanceMemorySize < targetInstanceMemorySize {
		return nil, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("target instance memory size must be lower than current instance memory size"),
		)
	}

	if currentInstanceCPUSize < targetInstanceCPUSize {
		return nil, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("target instance cpu size must be lower than current instance cpu size"),
		)
	}

	return &downsizeRequestParams{
		currentInstanceMemorySize: currentInstanceMemorySize,
		currentInstanceCPUSize:    currentInstanceCPUSize,
		targetInstanceMemorySize:  targetInstanceMemorySize,
		targetInstanceCPUSize:     targetInstanceCPUSize,
		numControlPlaneNodes:      numControlPlaneNodes,
	}, nil
}

func (d downsizeRequestParams) GetInstanceMemorySizeGB() int64 {
	return d.currentInstanceMemorySize
}

func (d downsizeRequestParams) GetInstanceCPUSize() int64 {
	return d.currentInstanceCPUSize
}

func (d downsizeRequestParams) GetTargetInstanceMemorySizeGB() int64 {
	return d.targetInstanceMemorySize
}

func (d downsizeRequestParams) GetTargetInstanceCPUSize() int64 {
	return d.targetInstanceCPUSize
}

func (d downsizeRequestParams) GetNumControlPlaneNodes() int64 {
	return d.numControlPlaneNodes
}
