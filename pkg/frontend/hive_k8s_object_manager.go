package frontend

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/Azure/ARO-RP/pkg/env"
)

type HiveK8sObjectManager interface {
	List(ctx context.Context, region, resource string) ([]byte, error)
	Get(ctx context.Context, region, resource, name string) ([]byte, error)
}

type hiveK8sObjectManager struct {
	env env.Interface
}

func (m *hiveK8sObjectManager) Get(
	ctx context.Context,
	region string,
	resource string,
	name string,
) ([]byte, error) {

	// 1. Get Hive REST config
	restConfig, err := m.env.LiveConfig().HiveRestConfig(ctx, 1)
	if err != nil {
		return nil, err
	}

	// 2. Create dynamic client
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	// 3. Resolve GVR (same logic as List)
	gvr := schema.GroupVersionResource{
		Group:    "",   // fill correctly
		Version:  "v1", // fill correctly
		Resource: resource,
	}

	// 4. Get object
	u, err := dyn.Resource(gvr).
		Namespace("default"). // or namespace param if supported
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return json.Marshal(u)
}
