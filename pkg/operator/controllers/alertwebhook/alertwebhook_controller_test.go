package alertwebhook

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
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
  - url: http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080/healthz/ready
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
  - url: http://aro-operator-master.openshift-azure-operator.svc.cluster.local:8080/healthz/ready
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
		name              string
		alertmanagerYaml  []byte
		controllerEnabled bool
		want              []byte
	}{
		{
			name:              "old cluster, enabled",
			alertmanagerYaml:  initialOld,
			controllerEnabled: true,
			want:              wantOld,
		},
		{
			name:              "new cluster, enabled",
			alertmanagerYaml:  initialNew,
			controllerEnabled: true,
			want:              wantNew,
		},
		{
			name:              "old cluster, disabled",
			alertmanagerYaml:  initialOld,
			controllerEnabled: false,
			want:              initialOld,
		},
		{
			name:              "new cluster, disabled",
			alertmanagerYaml:  initialNew,
			controllerEnabled: false,
			want:              initialNew,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Spec: arov1alpha1.ClusterSpec{
					OperatorFlags: arov1alpha1.OperatorFlags{
						operator.AlertWebhookEnabled: operator.FlagFalse,
					},
				},
			}

			if tt.controllerEnabled {
				instance.Spec.OperatorFlags[operator.AlertWebhookEnabled] = operator.FlagTrue
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alertmanager-main",
					Namespace: "openshift-monitoring",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": tt.alertmanagerYaml,
				},
			}

			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				client: ctrlfake.NewClientBuilder().WithObjects(instance, secret).Build(),
			}

			_, err := r.Reconcile(ctx, ctrl.Request{})
			if err != nil {
				t.Fatal(err)
			}

			s := &corev1.Secret{}
			err = r.client.Get(ctx, types.NamespacedName{Namespace: "openshift-monitoring", Name: "alertmanager-main"}, s)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(bytes.Trim(s.Data["alertmanager.yaml"], "\n"), bytes.Trim(tt.want, "\n")) {
				t.Error(string(s.Data["alertmanager.yaml"]))
			}
		})
	}
}
