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

const (
	etcdKeyCountContainer = "etcdctl"

	// etcdKeyCountCommand lists all etcd keys under /kubernetes.io/registry/, groups
	// them by namespace (the fifth path segment), and returns the top 10 by count.
	// OpenShift etcd stores namespaced resources as /kubernetes.io/registry/RESOURCE/NAMESPACE/NAME
	// (6 segments when split by '/'), so NF >= 6 selects only namespaced resources and
	// $5 is the namespace. The fixed command ensures callers cannot inject arbitrary
	// shell commands through this endpoint.
	etcdKeyCountCommand = `etcdctl get --prefix /kubernetes.io/registry/ --keys-only | ` +
		`awk -F'/' 'NF >= 6 {count[$5]++} END {for (ns in count) print count[ns], ns}' | ` +
		`sort -nr | head -n 10`
)

func (f *frontend) postAdminOpenShiftClusterEtcdKeyCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
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

	nodeName := r.URL.Query().Get("nodeName")
	if nodeName == "" || !rxKubernetesString.MatchString(nodeName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "",
			fmt.Sprintf("The provided nodeName '%s' is invalid.", nodeName))
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
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	podName := "etcd-" + nodeName
	go execContainerStream(ctx, k, namespaceEtcds, podName, etcdKeyCountContainer, etcdKeyCountCommand, writer)
	return nil
}
