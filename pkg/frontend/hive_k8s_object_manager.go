package frontend

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
)

// HiveK8sObjectManager defines LIST / GET behavior for Hive-managed AKS clusters
type HiveK8sObjectManager interface {
	List(ctx context.Context, resource, namespace string) ([]byte, error)
	Get(ctx context.Context, resource, namespace, name string) ([]byte, error)
}

type hiveK8sObjectManager struct {
	env                env.Interface
	kubeActionsFactory kubeActionsFactory
}

// constructor used by frontend
func newHiveK8sObjectManager(
	env env.Interface,
	kubeActionsFactory kubeActionsFactory,
) HiveK8sObjectManager {
	return &hiveK8sObjectManager{
		env:                env,
		kubeActionsFactory: kubeActionsFactory,
	}
}

// List Kubernetes objects in a Hive-managed AKS cluster
func (m *hiveK8sObjectManager) List(
	ctx context.Context,
	resource string,
	namespace string,
) ([]byte, error) {
	log := logrus.NewEntry(logrus.StandardLogger())

	// Reuse existing kube actions (same as other admin actions)
	k, err := m.kubeActionsFactory(log, m.env, nil)
	if err != nil {
		return nil, err
	}

	// Borrow existing GVR resolution behavior
	_, err = k.ResolveGVR(resource, "")
	if err != nil {
		return nil, err
	}

	// Delegate to existing list behavior
	return k.KubeList(ctx, resource, namespace)
}

// Get a single Kubernetes object in a Hive-managed AKS cluster
func (m *hiveK8sObjectManager) Get(
	ctx context.Context,
	resource string,
	namespace string,
	name string,
) ([]byte, error) {
	log := logrus.NewEntry(logrus.StandardLogger())

	k, err := m.kubeActionsFactory(log, m.env, nil)
	if err != nil {
		return nil, err
	}

	// Borrow existing GVR resolution behavior
	_, err = k.ResolveGVR(resource, "")
	if err != nil {
		return nil, err
	}

	return k.KubeGet(ctx, resource, namespace, name)
}
