package frontend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

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
		b, err = f.getHiveK8sObject(ctx, resource, namespace, name)
	} else {
		b, err = f.listHiveK8sObjects(ctx, resource, namespace)
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) listHiveK8sObjects(ctx context.Context, resource, namespace string) ([]byte, error) {
	if f.kubeActionsFactory == nil {
		return nil, fmt.Errorf("kube actions factory not configured")
	}

	log := logrus.NewEntry(logrus.StandardLogger())

	k, err := f.kubeActionsFactory(log, f.env, nil)
	if err != nil {
		return nil, err
	}

	_, err = k.ResolveGVR(resource, "")
	if err != nil {
		return nil, err
	}

	return k.KubeList(ctx, resource, namespace)
}

func (f *frontend) getHiveK8sObject(ctx context.Context, resource, namespace, name string) ([]byte, error) {
	if f.kubeActionsFactory == nil {
		return nil, fmt.Errorf("kube actions factory not configured")
	}

	log := logrus.NewEntry(logrus.StandardLogger())

	k, err := f.kubeActionsFactory(log, f.env, nil)
	if err != nil {
		return nil, err
	}

	_, err = k.ResolveGVR(resource, "")
	if err != nil {
		return nil, err
	}

	return k.KubeGet(ctx, resource, namespace, name)
}
