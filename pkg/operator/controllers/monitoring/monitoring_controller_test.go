package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

var (
	cmMetadata = metav1.ObjectMeta{Name: "cluster-monitoring-config", Namespace: "openshift-monitoring"}
)

func TestReconcileMonitoringConfig(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	type test struct {
		name       string
		configMap  *corev1.ConfigMap
		wantConfig string
	}

	for _, tt := range []*test{
		{
			name:       "ConfigMap does not exist - enable",
			wantConfig: `{}`,
		},
		{
			name: "empty config.yaml",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"config.yaml": ``,
				},
			},
			wantConfig: ``,
		},
		{
			name: "settings restored to default and extra fields are preserved",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"config.yaml": `
prometheusK8s:
  extraField: prometheus
  retention: 1d
  volumeClaimTemplate:
    metadata:
      name: meh
    spec:
      resources:
        requests:
          storage: 50Gi
      storageClassName: fast
      volumeMode: Filesystem
alertmanagerMain:
  extraField: yeet
  volumeClaimTemplate:
    metadata:
      name: slowest-storage
    spec:
      resources:
        requests:
          storage: 50Gi
        storageClassName: snail-mail
        volumeMode: Filesystem
`,
				},
			},
			wantConfig: `
alertmanagerMain:
  extraField: yeet
prometheusK8s:
  extraField: prometheus
`,
		},
		{
			name: "empty volumeClaimTemplate struct is cleared out",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"config.yaml": `
alertmanagerMain:
  volumeClaimTemplate: {}
  extraField: alertmanager
prometheusK8s:
  volumeClaimTemplate: {}
  bugs: not-here
`,
				},
			},
			wantConfig: `
alertmanagerMain:
  extraField: alertmanager
prometheusK8s:
  bugs: not-here
`,
		},
		{
			name: "other monitoring components are configured",
			configMap: &corev1.ConfigMap{
				ObjectMeta: cmMetadata,
				Data: map[string]string{
					"config.yaml": `
alertmanagerMain:
  nodeSelector:
    foo: bar
somethingElse:
  configured: true
`,
				},
			},
			wantConfig: `
alertmanagerMain:
  nodeSelector:
    foo: bar
somethingElse:
  configured: true
`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
					},
				},
			}

			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(instance)
			if tt.configMap != nil {
				clientBuilder.WithObjects(tt.configMap)
			}

			r := &Reconciler{
				log:        log,
				client:     clientBuilder.Build(),
				jsonHandle: new(codec.JsonHandle),
			}
			request := ctrl.Request{}
			request.Name = "cluster-monitoring-config"
			request.Namespace = "openshift-monitoring"

			_, err := r.Reconcile(ctx, request)
			if err != nil {
				t.Fatal(err)
			}

			cm := &corev1.ConfigMap{}
			err = r.client.Get(ctx, types.NamespacedName{Namespace: "openshift-monitoring", Name: "cluster-monitoring-config"}, cm)
			if err != nil {
				t.Fatal(err)
			}

			if strings.TrimSpace(cm.Data["config.yaml"]) != strings.TrimSpace(tt.wantConfig) {
				t.Error(cm.Data["config.yaml"])
			}
		})
	}
}

func TestReconcilePVC(t *testing.T) {
	volumeMode := corev1.PersistentVolumeFilesystem
	tests := []struct {
		name string
		pvcs []client.Object
		want []corev1.PersistentVolumeClaim
	}{
		{
			name: "Should delete the prometheus PVCs",
			pvcs: []client.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-k8s-db-prometheus-k8s-0",
						Namespace: "openshift-monitoring",
						Labels: map[string]string{
							"app":        "prometheus",
							"prometheus": "k8s",
						},
					},
				},
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-k8s-db-prometheus-k8s-1",
						Namespace: "openshift-monitoring",
						Labels: map[string]string{
							"app":        "prometheus",
							"prometheus": "k8s",
						},
					},
				},
			},
			want: []corev1.PersistentVolumeClaim{},
		},
		{
			name: "Should preserve 1 pvc",
			pvcs: []client.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-k8s-db-prometheus-k8s-0",
						Namespace: "openshift-monitoring",
						Labels: map[string]string{
							"app":        "prometheus",
							"prometheus": "k8s",
						},
					},
				},
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "random-pvc",
						Namespace: "openshift-monitoring",
						Labels: map[string]string{
							"app": "random",
						},
						ResourceVersion: "1",
					},
				},
			},
			want: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "random-pvc",
						Namespace: "openshift-monitoring",
						Labels: map[string]string{
							"app": "random",
						},
						ResourceVersion: "1",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeMode: &volumeMode,
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimPending,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						controllerEnabled: "true",
					},
				},
			}

			clientFake := ctrlfake.NewClientBuilder().WithObjects(instance).WithObjects(tt.pvcs...).Build()

			r := &Reconciler{
				log:        logrus.NewEntry(logrus.StandardLogger()),
				client:     clientFake,
				jsonHandle: new(codec.JsonHandle),
			}
			request := ctrl.Request{}
			request.Name = "cluster-monitoring-config"
			request.Namespace = "openshift-monitoring"

			_, err := r.Reconcile(ctx, request)
			if err != nil {
				t.Fatal(err)
			}

			pvcList := &corev1.PersistentVolumeClaimList{}
			err = r.client.List(ctx, pvcList, &client.ListOptions{
				Namespace: monitoringName.Namespace,
			})
			if err != nil {
				t.Fatalf("Unexpected error during list of PVCs: %v", err)
			}

			if !reflect.DeepEqual(pvcList.Items, tt.want) {
				t.Error(cmp.Diff(pvcList.Items, tt.want))
			}
		})
	}
}
