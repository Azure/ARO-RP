package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenShiftClusterVMResize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	err := f._postAdminOpenShiftClusterVMResize(log, ctx, r)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterVMResize(log *logrus.Entry, ctx context.Context, r *http.Request) error {
	vars := mux.Vars(r)
	vmName := r.URL.Query().Get("vmName")
	action, doc, err := f.prepareAdminActions(log, ctx, vmName, strings.TrimPrefix(r.URL.Path, "/admin"), vars)
	if err != nil {
		return err
	}

	vmSize := r.URL.Query().Get("vmSize")
	err = validateAdminMasterVMSize(vmSize)
	if err != nil {
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	nodeList, err := k.KubeList(ctx, "node", "")
	if err != nil {
		return err
	}

	var u unstructured.Unstructured
	var nodes corev1.NodeList
	if err = json.Unmarshal(nodeList, &u); err != nil {
		return err
	}

	err = kruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &nodes)
	if err != nil {
		return err
	}

	nodeExists := false
	for _, node := range nodes.Items {
		if _, ok := node.ObjectMeta.Labels["node-role.kubernetes.io/master"]; !ok {
			continue
		}

		if strings.EqualFold(vmName, node.ObjectMeta.Name) {
			nodeExists = true
			break
		}
	}

	if !nodeExists {
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "",
			`"The master node '%s' under resource group '%s' was not found."`,
			vmName, vars["resourceGroupName"])
	}

	return action.VMResize(ctx, vmName, vmSize)
}
