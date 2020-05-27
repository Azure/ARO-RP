package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	initial = []byte(`
"global":
  "resolve_timeout": "5m"
"receivers":
- "name": "null"
"route":
  "group_by":
  - "job"
  "group_interval": "5m"
  "group_wait": "30s"
  "receiver": "null"
  "repeat_interval": "12h"
  "routes":
  - "match":
      "alertname": "Watchdog"
    "receiver": "null"
`)

	want = []byte(`
global:
  resolve_timeout: 5m
receivers:
- name: "null"
  webhook_configs:
  - url: http://localhost:1234/
route:
  group_by:
  - job
  group_interval: 5m
  group_wait: 30s
  receiver: "null"
  repeat_interval: 12h
  routes:
  - match:
      alertname: Watchdog
    receiver: "null"
`)
)

func TestSetAlertManagerWebhook(t *testing.T) {
	i := &AlertWebhookReconciler{
		Kubernetescli: fake.NewSimpleClientset(&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "alertmanager-main",
				Namespace: "openshift-monitoring",
			},
			Data: map[string][]byte{
				"alertmanager.yaml": initial,
			},
		}),
	}

	err := i.setAlertManagerWebhook("http://localhost:1234/")
	if err != nil {
		t.Fatal(err)
	}

	s, err := i.Kubernetescli.CoreV1().Secrets("openshift-monitoring").Get("alertmanager-main", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(bytes.Trim(s.Data["alertmanager.yaml"], "\n"), bytes.Trim(want, "\n")) {
		t.Error(string(s.Data["alertmanager.yaml"]))
	}
}
