package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminHiveClusterDeployment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceID := strings.TrimPrefix(filepath.Dir(r.URL.Path), "/admin")
	b, err := f._getAdminHiveClusterDeployment(ctx, resourceID)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminHiveClusterDeployment(ctx context.Context, resourceID string) ([]byte, error) {
	// we have to check if the frontend has a valid clustermanager since hive is not everywhere.
	if f.hiveClusterManager == nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "hive is not enabled")
	}

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "cluster not found")
	}

	if doc.OpenShiftCluster.Properties.HiveProfile.Namespace == "" {
		return nil, api.NewCloudError(http.StatusNoContent, api.CloudErrorCodeResourceNotFound, "", "cluster is not managed by hive")
	}

	cd, err := f.hiveClusterManager.GetClusterDeployment(ctx, doc)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "cluster deployment not found")
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, &codec.JsonHandle{}).Encode(cd)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unable to marshal response")
	}

	return b, nil
}
