package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func getTestSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azure-cloud-provider",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"cloud-config": []byte(`aadClientId: foo
aadClientSecret: bar`),
		},
	}
}

func getTestConfigMap(clientID, secret string) *corev1.ConfigMap {
	config := map[string]interface{}{
		"aadClientId":     clientID,
		"aadClientSecret": secret,
		"otherKey":        "value",
	}

	b, _ := json.MarshalIndent(config, "", "\t")

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-provider-config",
			Namespace: "openshift-config",
		},
		Data: map[string]string{
			"config": string(b),
		},
	}
}

func TestFixCloudConfig(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    *corev1.ConfigMap
		expected *corev1.ConfigMap
	}{
		{
			name:     "enrich",
			input:    getTestConfigMap("", ""),
			expected: getTestConfigMap("foo", "bar"),
		},
		{
			name:     "skip",
			input:    getTestConfigMap("", "bar"),
			expected: getTestConfigMap("", "bar"),
		},
	} {
		i := &Installer{
			kubernetescli: k8sfake.NewSimpleClientset(getTestSecret(), tt.input),
			log:           logrus.NewEntry(logrus.StandardLogger()),
		}
		err := i.fixCloudConfig(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		result, err := i.kubernetescli.CoreV1().ConfigMaps("openshift-config").Get("cloud-provider-config", metav1.GetOptions{})
		if err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(tt.expected, result) {
			t.Error(cmp.Diff(tt.expected, result))
		}
	}
}
