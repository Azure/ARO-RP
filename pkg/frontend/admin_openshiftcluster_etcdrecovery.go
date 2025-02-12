package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

func (f *frontend) postAdminOpenShiftClusterEtcdRecovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._postAdminOpenShiftClusterEtcdRecovery(ctx, r, log)

	if err == nil {
		w.Header().Set("Content-Type", "text/plain")
	}

	adminReply(log, w, nil, b, err)
}

// TODO write integration test that skips f.fixEtcd
func (f *frontend) _postAdminOpenShiftClusterEtcdRecovery(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return []byte{}, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return []byte{}, err
	}
	kubeActions, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return []byte{}, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	gvr, err := kubeActions.ResolveGVR("Etcd", "")
	if err != nil {
		return []byte{}, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	err = validateAdminKubernetesObjects(r.Method, gvr, namespaceEtcds, "cluster")
	if err != nil {
		return []byte{}, err
	}

	restConfig, err := restconfig.RestConfig(f.env, doc.OpenShiftCluster)
	if err != nil {
		return []byte{}, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	operatorcli, err := operatorclient.NewForConfig(restConfig)
	if err != nil {
		return []byte{}, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return f.fixEtcd(ctx, log, f.env, doc, kubeActions, operatorcli.OperatorV1().Etcds())
}
