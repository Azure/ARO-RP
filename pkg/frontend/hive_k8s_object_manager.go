package frontend

import "context"

type HiveK8sObjectManager interface {
	List(ctx context.Context, region, namespace, resource string) ([]byte, error)
}
