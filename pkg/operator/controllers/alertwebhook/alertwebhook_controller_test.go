package alertwebhook

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	initialOld = []byte(`
"global":
  "resolve_timeout": "5m"
"receivers":
- "name": "null"
"route":
  "group_by":
  - "namespace"
  "group_interval": "5m"
  "group_wait": "30s"
  "receiver": "null"
  "repeat_interval": "12h"
  "routes":
  - "match":
      "alertname": "Watchdog"
    "receiver": "null"
`)

	wantOld = []byte(`
global:
  resolve_timeout: 5m
receivers:
- name: "null"
  webhook_configs:
  - url: http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080
route:
  group_by:
  - namespace
  group_interval: 5m
  group_wait: 30s
  receiver: "null"
  repeat_interval: 12h
  routes:
  - match:
      alertname: Watchdog
    receiver: "null"
`)

	initialNew = []byte(`
"global":
  "resolve_timeout": "5m"
"inhibit_rules":
- "equal":
  - "namespace"
  - "alertname"
  "source_match":
    "severity": "critical"
  "target_match_re":
    "severity": "warning|info"
- "equal":
  - "namespace"
  - "alertname"
  "source_match":
    "severity": "warning"
  "target_match_re":
    "severity": "info"
"receivers":
- "name": "Default"
- "name": "Watchdog"
- "name": "Critical"
"route":
  "group_by":
  - "namespace"
  "group_interval": "5m"
  "group_wait": "30s"
  "receiver": "Default"
  "repeat_interval": "12h"
  "routes":
  - "match":
      "alertname": "Watchdog"
    "receiver": "Watchdog"
  - "match":
      "severity": "critical"
    "receiver": "Critical"
`)

	wantNew = []byte(`
global:
  resolve_timeout: 5m
inhibit_rules:
- equal:
  - namespace
  - alertname
  source_match:
    severity: critical
  target_match_re:
    severity: warning|info
- equal:
  - namespace
  - alertname
  source_match:
    severity: warning
  target_match_re:
    severity: info
receivers:
- name: Default
  webhook_configs:
  - url: http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080
- name: Watchdog
- name: Critical
route:
  group_by:
  - namespace
  group_interval: 5m
  group_wait: 30s
  receiver: Default
  repeat_interval: 12h
  routes:
  - match:
      alertname: Watchdog
    receiver: Watchdog
  - match:
      severity: critical
    receiver: Critical
`)
)

func TestSetAlertManagerWebhook(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		reconciler *Reconciler
		want       []byte
	}{
		{
			name: "old cluster",
			reconciler: &Reconciler{
				kubernetescli: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alertmanager-main",
						Namespace: "openshift-monitoring",
					},
					Data: map[string][]byte{
						"alertmanager.yaml": initialOld,
					},
				}),
			},
			want: wantOld,
		},
		{
			name: "new cluster",
			reconciler: &Reconciler{
				kubernetescli: fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "alertmanager-main",
						Namespace: "openshift-monitoring",
					},
					Data: map[string][]byte{
						"alertmanager.yaml": initialNew,
					},
				}),
			},
			want: wantNew,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := tt.reconciler

			err := i.setAlertManagerWebhook(ctx, "http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080")
			if err != nil {
				t.Fatal(err)
			}

			s, err := i.kubernetescli.CoreV1().Secrets("openshift-monitoring").Get(ctx, "alertmanager-main", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(bytes.Trim(s.Data["alertmanager.yaml"], "\n"), bytes.Trim(tt.want, "\n")) {
				t.Error(string(s.Data["alertmanager.yaml"]))
			}
		})
	}
}
