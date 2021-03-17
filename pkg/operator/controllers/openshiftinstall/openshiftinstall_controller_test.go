package openshiftinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var aroInvoker string = "ARO"

func TestSetOpenshiftInstallCM(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		reconciler *OpenshiftInstallReconciler
		want       string
	}{
		{
			name: "normal case",
			reconciler: &OpenshiftInstallReconciler{
				kubernetescli: fake.NewSimpleClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "openshift-install-manifests",
						Namespace: "openshift-config",
					},
					Data: map[string]string{
						"invoker": "user",
					},
				}),
			},
			want: aroInvoker,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := tt.reconciler

			aroInvoker := "ARO"

			err := i.setOpenshiftInstallInvoker(ctx, aroInvoker)
			if err != nil {
				t.Fatal(err)
			}

			cm, err := i.kubernetescli.CoreV1().ConfigMaps("openshift-config").Get(ctx, "openshift-install-manifests", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if cm.Data["invoker"] != tt.want {
				t.Error(tt.name + ": openshift-install-manifests .data.invoker is not " + aroInvoker)
			}
		})
	}
}
