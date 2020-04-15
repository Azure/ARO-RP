package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"regexp"
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

	b, err := f._getAdminKubernetesObjects(ctx, r)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminKubernetesObjects(ctx context.Context, r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)

	kind, namespace, name := r.URL.Query().Get("kind"), r.URL.Query().Get("namespace"), r.URL.Query().Get("name")

	err := validateAdminKubernetesObjects(r.Method, kind, namespace, name)
	if err != nil {
		return nil, err
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	if name != "" {
		return f.kubeActions.Get(ctx, doc.OpenShiftCluster, kind, namespace, name)
	}
	return f.kubeActions.List(ctx, doc.OpenShiftCluster, kind, namespace)
}

func (f *frontend) deleteAdminKubernetesObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._deleteAdminKubernetesObjects(ctx, r)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _deleteAdminKubernetesObjects(ctx context.Context, r *http.Request) error {
	vars := mux.Vars(r)

	kind, namespace, name := r.URL.Query().Get("kind"), r.URL.Query().Get("namespace"), r.URL.Query().Get("name")

	err := validateAdminKubernetesObjectsNonCustomer(r.Method, kind, namespace, name)
	if err != nil {
		return err
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return err
	}

	return f.kubeActions.Delete(ctx, doc.OpenShiftCluster, kind, namespace, name)
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

	err := f._postAdminKubernetesObjects(ctx, r)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminKubernetesObjects(ctx context.Context, r *http.Request) error {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	vars := mux.Vars(r)

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return err
	}

	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(body)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	err = validateAdminKubernetesObjectsNonCustomer(r.Method, obj.GroupVersionKind().GroupKind().String(), obj.GetNamespace(), obj.GetName())
	if err != nil {
		return err
	}

	return f.kubeActions.CreateOrUpdate(ctx, doc.OpenShiftCluster, obj)
}

// rxKubernetesString is weaker than Kubernetes validation, but strong enough to
// prevent mischief
var rxKubernetesString = regexp.MustCompile(`(?i)^[-a-z0-9]{0,255}$`)

func validateAdminKubernetesObjectsNonCustomer(method, kind, namespace, name string) error {
	if namespace != "" &&
		namespace != "default" &&
		namespace != "openshift" &&
		!strings.HasPrefix(string(namespace), "kube-") &&
		!strings.HasPrefix(string(namespace), "openshift-") {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to the provided namespace '%s' is forbidden.", namespace)
	}

	return validateAdminKubernetesObjects(method, kind, namespace, name)
}

func validateAdminKubernetesObjects(method, kind, namespace, name string) error {
	if kind == "" ||
		!rxKubernetesString.MatchString(kind) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided kind '%s' is invalid.", kind)
	}
	if strings.EqualFold(kind, "secret") {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to secrets is forbidden.")
	}

	if !rxKubernetesString.MatchString(namespace) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided namespace '%s' is invalid.", namespace)
	}

	if (method != http.MethodGet && name == "") ||
		!rxKubernetesString.MatchString(name) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided name '%s' is invalid.", name)
	}

	return nil
}
