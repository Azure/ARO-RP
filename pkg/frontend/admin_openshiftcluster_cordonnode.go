package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

var nodeResource = schema.GroupVersionResource{
	Group:    "",
	Resource: "nodes",
}

func (f *frontend) postAdminOpenShiftClusterCordonNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._postAdminOpenShiftClusterCordonNode(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterCordonNode(ctx context.Context, r *http.Request, log *logrus.Entry) error {
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]

	vmName := r.URL.Query().Get("vmName")
	shouldCordon := strings.EqualFold(r.URL.Query().Get("shouldCordon"), "true")
	err := validateAdminKubernetesObjects(r.Method, &nodeResource, "", vmName)
	if err != nil {
		return err
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	case err != nil:
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	return k.CordonNode(ctx, vmName, shouldCordon)
}
