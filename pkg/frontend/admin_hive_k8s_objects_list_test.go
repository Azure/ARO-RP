// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
package frontend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

//
// --- Test: factory nil error ---
//

func TestListHiveK8sObjects_NoFactory(t *testing.T) {
	f := &frontend{
		kubeActionsFactory: nil,
	}

	_, err := f.listHiveK8sObjects(context.Background(), "pods", "default")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestGetHiveK8sObject_NoFactory(t *testing.T) {
	f := &frontend{
		kubeActionsFactory: nil,
	}

	_, err := f.getHiveK8sObject(context.Background(), "pods", "default", "mypod")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

//
// verifies that name parameter controls list vs get behavior
//

func TestAdminHiveK8sObjectsList_NameControlsFlow(t *testing.T) {
	listCalled := false
	getCalled := false

	f := &frontend{
		kubeActionsFactory: func(log *logrus.Entry, e env.Interface, oc *api.OpenShiftCluster) (adminactions.KubeActions, error) {
			return &testKubeActions{
				listFn: func(ctx context.Context, resource, namespace string) ([]byte, error) {
					listCalled = true
					return []byte("{}"), nil
				},
				getFn: func(ctx context.Context, resource, namespace, name string) ([]byte, error) {
					getCalled = true
					return []byte("{}"), nil
				},
			}, nil
		},
	}

	// LIST path
	req := httptest.NewRequest(http.MethodGet, "/admin/hive/k8s/pods?namespace=default", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("resource", "pods")

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger()))
	req = req.WithContext(ctx)

	f.adminHiveK8sObjectsList(w, req)

	if !listCalled {
		t.Fatal("expected listHiveK8sObjects to be called")
	}
	if getCalled {
		t.Fatal("did not expect getHiveK8sObject to be called")
	}

	// reset
	listCalled = false
	getCalled = false

	// GET path
	req = httptest.NewRequest(http.MethodGet, "/admin/hive/k8s/pods?namespace=default&name=testpod", nil)
	w = httptest.NewRecorder()

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("resource", "pods")

	ctx = context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.ContextKeyLog, logrus.NewEntry(logrus.StandardLogger()))
	req = req.WithContext(ctx)

	f.adminHiveK8sObjectsList(w, req)

	if !getCalled {
		t.Fatal("expected getHiveK8sObject to be called")
	}
	if listCalled {
		t.Fatal("did not expect listHiveK8sObjects to be called")
	}
}

//
// verifies factory is called and receives oc == nil
//

func TestListHiveK8sObjects_FactoryCalledWithNilOC(t *testing.T) {
	called := false

	f := &frontend{
		kubeActionsFactory: func(log *logrus.Entry, e env.Interface, oc *api.OpenShiftCluster) (adminactions.KubeActions, error) {
			called = true

			if oc != nil {
				t.Fatal("expected oc to be nil")
			}

			return nil, fmt.Errorf("stop")
		},
	}

	_, _ = f.listHiveK8sObjects(context.Background(), "pods", "default")

	if !called {
		t.Fatal("expected kubeActionsFactory to be called")
	}
}

func TestGetHiveK8sObject_FactoryCalledWithNilOC(t *testing.T) {
	called := false

	f := &frontend{
		kubeActionsFactory: func(log *logrus.Entry, e env.Interface, oc *api.OpenShiftCluster) (adminactions.KubeActions, error) {
			called = true

			if oc != nil {
				t.Fatal("expected oc to be nil")
			}

			return nil, fmt.Errorf("stop")
		},
	}

	_, _ = f.getHiveK8sObject(context.Background(), "pods", "default", "mypod")

	if !called {
		t.Fatal("expected kubeActionsFactory to be called")
	}
}

//
// --- Mock: fully implements adminactions.KubeActions ---
//

type testKubeActions struct {
	listFn func(ctx context.Context, resource, namespace string) ([]byte, error)
	getFn  func(ctx context.Context, resource, namespace, name string) ([]byte, error)
}

func (t *testKubeActions) KubeGet(ctx context.Context, groupKind, namespace, name string) ([]byte, error) {
	if t.getFn != nil {
		return t.getFn(ctx, groupKind, namespace, name)
	}
	return []byte("{}"), nil
}

func (t *testKubeActions) KubeList(ctx context.Context, groupKind, namespace string) ([]byte, error) {
	if t.listFn != nil {
		return t.listFn(ctx, groupKind, namespace)
	}
	return []byte("{}"), nil
}

func (t *testKubeActions) KubeCreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error {
	return nil
}

func (t *testKubeActions) KubeDelete(ctx context.Context, groupKind, namespace, name string, force bool, propagationPolicy *metav1.DeletionPropagation) error {
	return nil
}

func (t *testKubeActions) ResolveGVR(groupKind string, optionalVersion string) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, nil
}

func (t *testKubeActions) CordonNode(ctx context.Context, nodeName string, unschedulable bool) error {
	return nil
}

func (t *testKubeActions) DrainNode(ctx context.Context, nodeName string) error {
	return nil
}

func (t *testKubeActions) ApproveCsr(ctx context.Context, csrName string) error {
	return nil
}

func (t *testKubeActions) ApproveAllCsrs(ctx context.Context) error {
	return nil
}

func (t *testKubeActions) KubeGetPodLogs(ctx context.Context, namespace, name, containerName string) ([]byte, error) {
	return []byte(""), nil
}

func (t *testKubeActions) KubeWatch(ctx context.Context, o *unstructured.Unstructured, label string) (watch.Interface, error) {
	return nil, nil
}

func (t *testKubeActions) TopPods(ctx context.Context, restConfig *restclient.Config, allNamespaces bool) ([]adminactions.PodMetrics, error) {
	return nil, nil
}

func (t *testKubeActions) TopNodes(ctx context.Context, restConfig *restclient.Config) ([]adminactions.NodeMetrics, error) {
	return nil, nil
}
