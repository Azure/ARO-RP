package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminHiveClusterDeployment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceID := r.URL.Query().Get("resourceID")
	clusterDeploymentNamespace := r.URL.Query().Get("clusterDeploymentNamespace")
	b, err := f._getAdminHiveClusterDeployment(ctx, resourceID, clusterDeploymentNamespace)
	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminHiveClusterDeployment(ctx context.Context, resourceID string, clusterDeploymentNamespace string) ([]byte, error) {
	var doc *api.OpenShiftClusterDocument

	// we have to check if the frontend has a valid clustermanager since hive is not everywhere.
	if f.hiveClusterManager == nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "hive is not enabled")
	}
	if resourceID == "" && clusterDeploymentNamespace == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "Parameters resourceID '%s' clusterDeploymentNamespace '%s' are both empty, atleast one should have a valid value.", resourceID, clusterDeploymentNamespace)
	}
	// if resourceID is not null, fetch the clusterDeploymentNamespace using the resourceID
	// when parameteres resourceID and clusterDeploymentNamespace are not null, clusterDeploymentNamespace will be ignored
	if resourceID != "" {
		doc, err := f.dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
		if err != nil {
			return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "cluster '%s' not found", resourceID)
		}
		if doc.OpenShiftCluster.Properties.HiveProfile.Namespace == "" {
			return nil, api.NewCloudError(http.StatusNoContent, api.CloudErrorCodeResourceNotFound, "", "cluster '%s' is not managed by hive", resourceID)
		}
	} else {
		doc = &api.OpenShiftClusterDocument{
			OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					HiveProfile: api.HiveProfile{
						Namespace: clusterDeploymentNamespace,
					},
				},
			},
		}
	}

	cd, err := f.hiveClusterManager.GetClusterDeployment(ctx, doc)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "cluster deployment '%s' not found", clusterDeploymentNamespace)
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, &codec.JsonHandle{}).Encode(cd)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unable to marshal response")
	}

	return b, nil
}
