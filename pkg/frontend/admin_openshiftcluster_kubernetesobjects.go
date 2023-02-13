package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminKubernetesObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._getAdminKubernetesObjects(ctx, r, log)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminKubernetesObjects(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	var err error
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]

	groupKind, namespace, name := r.URL.Query().Get("kind"), r.URL.Query().Get("namespace"), r.URL.Query().Get("name")

	unrestricted := false
	if r.URL.Query().Has("unrestricted") {
		unrestricted, err = strconv.ParseBool(r.URL.Query().Get("unrestricted"))
		if err != nil {
			return nil, err
		}
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	case err != nil:
		return nil, err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	gvr, err := k.ResolveGVR(groupKind)
	if err != nil {
		return nil, err
	}

	if !unrestricted {
		err = validateAdminKubernetesObjectsNonCustomer(r.Method, gvr, namespace, name)
		if err != nil {
			return nil, err
		}
	}
	err = validateAdminKubernetesObjects(r.Method, gvr, namespace, name)
	if err != nil {
		return nil, err
	}

	if name != "" {
		return k.KubeGet(ctx, groupKind, namespace, name)
	}
	return k.KubeList(ctx, groupKind, namespace)
}

func (f *frontend) deleteAdminKubernetesObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._deleteAdminKubernetesObjects(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _deleteAdminKubernetesObjects(ctx context.Context, r *http.Request, log *logrus.Entry) error {
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]

	groupKind, namespace, name := r.URL.Query().Get("kind"), r.URL.Query().Get("namespace"), r.URL.Query().Get("name")
	force := strings.EqualFold(r.URL.Query().Get("force"), "true")

	if force {
		err := validateAdminKubernetesObjectsForceDelete(groupKind)
		if err != nil {
			return err
		}
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

	gvr, err := k.ResolveGVR(groupKind)
	if err != nil {
		return err
	}

	err = validateAdminKubernetesObjectsNonCustomer(r.Method, gvr, namespace, name)
	if err != nil {
		return err
	}

	return k.KubeDelete(ctx, groupKind, namespace, name, force)
}

func (f *frontend) postAdminKubernetesObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) == 0 || !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
		return
	}

	err := f._postAdminKubernetesObjects(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminKubernetesObjects(ctx context.Context, r *http.Request, log *logrus.Entry) error {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	case err != nil:
		return err
	}

	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(body)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	gvr, err := k.ResolveGVR(obj.GetKind())
	if err != nil {
		return err
	}

	err = validateAdminKubernetesObjectsNonCustomer(r.Method, gvr, obj.GetNamespace(), obj.GetName())
	if err != nil {
		return err
	}

	return k.KubeCreateOrUpdate(ctx, obj)
}
