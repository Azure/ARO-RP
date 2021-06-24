package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	ctrl "sigs.k8s.io/controller-runtime"
)

var cmMetadata = metav1.ObjectMeta{Name: "cluster-monitoring-config", Namespace: "openshift-monitoring"}

func TestReconcileMonitoringConfig(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	type test struct {
		name         string
		setConfigMap func() *Reconciler
		wantConfig   string
	}

	for _, tt := range []*test{
		{
			name: "ConfigMap does not exist - enable",
			setConfigMap: func() *Reconciler {
				return &Reconciler{
					kubernetescli: fake.NewSimpleClientset(),
					log:           log,
					jsonHandle:    new(codec.JsonHandle),
				}
			},
			wantConfig: `{}`,
		},
		{
			name: "empty config.yaml",
			setConfigMap: func() *Reconciler {
				return &Reconciler{
					kubernetescli: fake.NewSimpleClientset(&corev1.ConfigMap{
						ObjectMeta: cmMetadata,
						Data: map[string]string{
							"config.yaml": ``,
						},
					}),
					log:        log,
					jsonHandle: new(codec.JsonHandle),
				}
			},
			wantConfig: ``,
		},
		{
			name: "settings restored to default and extra fields are preserved",
			setConfigMap: func() *Reconciler {
				return &Reconciler{
					kubernetescli: fake.NewSimpleClientset(&corev1.ConfigMap{
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
					}),
					log:        log,
					jsonHandle: new(codec.JsonHandle),
				}
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
			setConfigMap: func() *Reconciler {
				return &Reconciler{
					kubernetescli: fake.NewSimpleClientset(&corev1.ConfigMap{
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
					}),
					log:        log,
					jsonHandle: new(codec.JsonHandle),
				}
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
			setConfigMap: func() *Reconciler {
				return &Reconciler{
					kubernetescli: fake.NewSimpleClientset(&corev1.ConfigMap{
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
					}),
					log:        log,
					jsonHandle: new(codec.JsonHandle),
				}
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
			r := tt.setConfigMap()
			request := ctrl.Request{}
			request.Name = "cluster-monitoring-config"
			request.Namespace = "openshift-monitoring"

			_, err := r.Reconcile(ctx, request)
			if err != nil {
				t.Fatal(err)
			}

			cm, err := r.kubernetescli.CoreV1().ConfigMaps("openshift-monitoring").Get(ctx, "cluster-monitoring-config", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if strings.TrimSpace(cm.Data["config.yaml"]) != strings.TrimSpace(tt.wantConfig) {
				t.Error(cm.Data["config.yaml"])
			}
		})
	}
}
