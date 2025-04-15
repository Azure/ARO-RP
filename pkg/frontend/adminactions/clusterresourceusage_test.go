package adminactions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func TestTopNodes_Unit(t *testing.T) {
	ctx := context.Background()

	kubecli := k8sfake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resourceMustParse("4"),
				corev1.ResourceMemory: resourceMustParse("8Gi"),
			},
		},
	})

	_ = metricsfake.NewSimpleClientset(&metricsv1beta1.NodeMetrics{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resourceMustParse("1000m"),
			corev1.ResourceMemory: resourceMustParse("2Gi"),
		},
	})

	k := &kubeActions{
		kubecli: kubecli,
	}

	result, err := k.TopNodes(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	node := result[0]
	assert.Equal(t, "node-1", node.NodeName)
	assert.Equal(t, "1000m", node.CPUUsage)
	assert.Equal(t, "2Gi", node.MemoryUsage)
	assert.InDelta(t, 25.0, node.CPUPercentage, 0.01)
	assert.InDelta(t, 25.0, node.MemoryPercentage, 0.01)
}

func resourceMustParse(val string) resource.Quantity {
	q, err := resource.ParseQuantity(val)
	if err != nil {
		panic(err)
	}
	return q
}
