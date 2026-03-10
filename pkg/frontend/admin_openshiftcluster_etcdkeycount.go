package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenShiftClusterEtcdKeyCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
	defer reader.Close()
	err := f._postAdminOpenShiftClusterEtcdKeyCount(ctx, r, log, writer)
	if err != nil {
		_ = writer.CloseWithError(err)
	}
	var header http.Header
	if err == nil {
		header = http.Header{"Content-Type": []string{"text/plain"}}
	}
	f.streamResponder.AdminReplyStream(log, w, header, reader, err)
}

func (f *frontend) _postAdminOpenShiftClusterEtcdKeyCount(ctx context.Context, r *http.Request, log *logrus.Entry, writer io.WriteCloser) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	vmName := r.URL.Query().Get("vmName")
	if vmName == "" || !rxKubernetesString.MatchString(vmName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided vmName '%s' is invalid.", vmName))
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return fmt.Errorf("fetching cluster document: %w", err)
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return fmt.Errorf("creating kube actions: %w", err)
	}

	log = log.WithFields(logrus.Fields{"resourceID": resourceID, "vmName": vmName})
	podName := "etcd-" + vmName
	opCtx, opCancel := context.WithCancel(ctx)
	go func() {
		defer opCancel()
		execContainerStream(opCtx, log, k, namespaceEtcds, podName, etcdContainerName, etcdKeyCountScript, writer)
	}()
	return nil
}
