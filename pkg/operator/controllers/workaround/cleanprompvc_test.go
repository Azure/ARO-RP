package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func TestCleanPromPVCEnsure(t *testing.T) {

	pvc0 := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-k8s-db-prometheus-k8s-00",
			Namespace: "openshift-monitoring",
			Labels: map[string]string{
				"app":        "prometheus",
				"prometheus": "k8s",
			},
		},
	}

	pvc1 := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-k8s-db-prometheus-k8s-0",
			Namespace: "openshift-monitoring",
			Labels: map[string]string{
				"app":        "prometheus",
				"prometheus": "k8s",
			},
		},
	}

	pvc2 := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-k8s-db-prometheus-k8s-1",
			Namespace: "openshift-monitoring",
			Labels: map[string]string{
				"app":        "prometheus",
				"prometheus": "k8s",
			},
		},
	}

	pvc3 := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: "openshift-monitoring",
		},
	}

	tests := []struct {
		name       string
		cli        *fake.Clientset
		wantPVCNum int
		wantErr    error
	}{
		{
			name:       "Should delete the prometheus PVCs",
			cli:        fake.NewSimpleClientset(&pvc1, &pvc2, &pvc3),
			wantPVCNum: 1,
			wantErr:    nil,
		},
		{
			name:       "Should not delete the prometheus PVCs, too many items",
			cli:        fake.NewSimpleClientset(&pvc1, &pvc2, &pvc3, &pvc0),
			wantPVCNum: 1,
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewCleanFromPVCWorkaround(utillog.GetLogger(), tt.cli)
			err := w.Ensure(context.Background())
			if err != tt.wantErr {
				t.Fatalf("Unexpected error\nwant: %v\ngot: %v", tt.wantErr, err)
			}

			pvcList, err := tt.cli.CoreV1().PersistentVolumeClaims(monitoringNamespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatalf("Unexpected error during list of PVCs: %v", err)
			}
			if len(pvcList.Items) != tt.wantPVCNum {
				t.Fatalf("Unexpected number of PVCs\nwant: %d\ngot: %d", tt.wantPVCNum, len(pvcList.Items))
			}
		})
	}
}

func TestCleanPromPVCIsRequired(t *testing.T) {
	newKubernetesCli := func(config string) *fake.Clientset {
		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      monitoringName,
				Namespace: monitoringNamespace,
			},
			Data: make(map[string]string),
		}

		configMap.Data["config.yaml"] = config

		return fake.NewSimpleClientset(&configMap)
	}

	tests := []struct {
		name         string
		kcli         *fake.Clientset
		wantRequired bool
	}{
		{
			name: "Should not be required, persistent set true",
			kcli: newKubernetesCli(`prometheusK8s:
  retention: 15d
  volumeClaimTemplate:
    spec:
      resources:
        requests:
          storage: 100Gi
            `),
			wantRequired: false,
		},
		{
			name: "Should be required, persistent set to false",
			kcli: newKubernetesCli(`prometheusK8s:
  retention: ""
  volumeClaimTemplate:
    spec:
      resources:
        requests:
          storage: ""
            `),
			wantRequired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewCleanFromPVCWorkaround(utillog.GetLogger(), tt.kcli)
			required := w.IsRequired(nil)
			if required != tt.wantRequired {
				t.Fatalf("Unexpected workaroud required result\nwant: %t\ngot: %t", tt.wantRequired, required)
			}

		})
	}
}
