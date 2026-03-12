package frontend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHiveK8sObjectManager_Creation(t *testing.T) {
	manager := newHiveK8sObjectManager(nil, nil)

	require.NotNil(t, manager)
}

func TestHiveK8sObjectManager_List_WithNilDependencies(t *testing.T) {
	manager := newHiveK8sObjectManager(nil, nil)

	ctx := context.Background()

	_, err := manager.List(ctx, "pods", "default")

	require.Error(t, err)
}

func TestHiveK8sObjectManager_Get_WithNilDependencies(t *testing.T) {
	manager := newHiveK8sObjectManager(nil, nil)

	ctx := context.Background()

	_, err := manager.Get(ctx, "pods", "default", "mypod")

	require.Error(t, err)
}
