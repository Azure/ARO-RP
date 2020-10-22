package cluster

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *manager) configureClusterMonitoring(ctx context.Context) error {
	// 15 days is cluster-monitoring-operator current default retention.
	// storage requrement can be found here:
	// https://docs.openshift.com/container-platform/4.4/scalability_and_performance/scaling-cluster-monitoring-operator.html#prometheus-database-storage-requirements_cluster-monitoring-operator
	configData := `prometheusK8s:
  retention: 15d
  volumeClaimTemplate:
    spec:
      storageClassName: managed-premium
      resources:
        requests:
          storage: 100Gi
`
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-monitoring-config",
			Namespace: "openshift-monitoring",
		},
		Data: map[string]string{
			"config.yaml": configData,
		},
	}

	_, err := m.kubernetescli.CoreV1().ConfigMaps("openshift-monitoring").Create(cm)
	return err
}
