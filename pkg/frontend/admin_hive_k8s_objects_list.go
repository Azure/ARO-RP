package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) adminHiveK8sObjectsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resource := chi.URLParam(r, "resource")
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")

	var (
		b   []byte
		err error
	)

	if name != "" {
		b, err = f.getHiveK8sObject(ctx, log, resource, namespace, name)
	} else {
		b, err = f.listHiveK8sObjects(ctx, log, resource, namespace)
	}

	adminReply(log, w, nil, b, err)
}

// getHiveClient builds a dynamic client pointed at the Hive AKS cluster.
// used by hiveClusterManager (pkg/hive/manager.go NewFromEnvCLusterManager).
func (f *frontend) getHiveDynamicClient(ctx context.Context) (dynamic.Interface, meta.RESTMapper, error) {
	if f.hiveClusterManager == nil {
		return nil, nil, fmt.Errorf("hive is not enabled")
	}

	hiveShard := 1
	restConfig, err := f.env.LiveConfig().HiveRestConfig(ctx, hiveShard)
	if err != nil {
		return nil, nil, fmt.Errorf("failed getting Hive REST config: %w", err)
	}

	mapper, err := apiutil.NewDynamicRESTMapper(restConfig, apiutil.WithLazyDiscovery)
	if err != nil {
		return nil, nil, err
	}

	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}

	return dyn, mapper, nil
}

func (f *frontend) listHiveK8sObjects(ctx context.Context, log *logrus.Entry, resource, namespace string) ([]byte, error) {
	dyn, mapper, err := f.getHiveDynamicClient(ctx)
	if err != nil {
		return nil, err
	}

	gvr, err := mapper.ResourceFor(schema.ParseGroupResource(resource).WithVersion(""))
	if err != nil {
		return nil, err
	}

	ul, err := dyn.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1000})
	if err != nil {
		return nil, err
	}

	return ul.MarshalJSON()
}

func (f *frontend) getHiveK8sObject(ctx context.Context, log *logrus.Entry, resource, namespace, name string) ([]byte, error) {
	dyn, mapper, err := f.getHiveDynamicClient(ctx)
	if err != nil {
		return nil, err
	}

	gvr, err := mapper.ResourceFor(schema.ParseGroupResource(resource).WithVersion(""))
	if err != nil {
		return nil, err
	}

	un, err := dyn.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return un.MarshalJSON()
}
