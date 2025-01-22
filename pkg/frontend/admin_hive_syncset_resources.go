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
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminHiveClusterSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceId := strings.TrimPrefix(filepath.Dir(r.URL.Path), "/admin")
	b, err := f._getAdminHiveClusterSync(ctx, resourceId)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminHiveClusterSync(ctx context.Context, resourceId string) ([]byte, error) {
	// We have to check if the frontend has a valid clustermanager since hive is not everywhere.
	if f.hiveClusterManager == nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "hive is not enabled")
	}

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceId)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", err.Error())
	case err != nil:
		return nil, err
	}

	if doc.OpenShiftCluster.Properties.HiveProfile.Namespace == "" {
		return nil, api.NewCloudError(http.StatusNoContent, api.CloudErrorCodeResourceNotFound, "", "cluster is not managed by hive")
	}

	cd, err := f.hiveClusterManager.GetClusterSync(ctx, doc)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", err.Error())
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, &codec.JsonHandle{}).Encode(cd)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return b, nil
}
