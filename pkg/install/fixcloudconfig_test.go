package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
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
			"cloud-config": []byte(`aadClientId: 123e4567-e89b-12d3-a456-426614174000
aadClientSecret: 123e4567-e89b-12d3-a456-426614174000`),
		},
	}
}

func getTestConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-provider-config",
			Namespace: "openshift-config",
		},
		Data: map[string]string{
			"config": `{"aadClientCertPath": "",
				"aadClientId": "",
				"aadClientSecret": ""
		}`,
		},
	}
}

func TestFixCloudConfig(t *testing.T) {
	for _, tt := range []struct {
		name      string
		configMap func() *corev1.ConfigMap
		modify    func(cm *corev1.ConfigMap)
	}{
		{
			name: "enrich",
			configMap: func() *corev1.ConfigMap {
				return getTestConfigMap()
			},
			modify: func(cm *corev1.ConfigMap) {
				if _, ok := cm.Data["config"]; ok {
					var config map[string]interface{}
					err := yaml.Unmarshal([]byte(cm.Data["config"]), &config)
					if err != nil {
						t.Error(err)
						t.FailNow()
					}
					config["aadClientId"] = "123e4567-e89b-12d3-a456-426614174000"
					config["aadClientSecret"] = "123e4567-e89b-12d3-a456-426614174000"

					b, err := json.MarshalIndent(config, "", "\t")
					if err != nil {
						t.Error(err)
						t.FailNow()
					}
					cm.Data["config"] = string(b)
				}
			},
		},
		{
			name: "skip",
			configMap: func() *corev1.ConfigMap {
				cm := getTestConfigMap()
				if _, ok := cm.Data["config"]; ok {
					var config map[string]interface{}
					err := yaml.Unmarshal([]byte(cm.Data["config"]), &config)
					if err != nil {
						t.Error(err)
						t.FailNow()
					}
					config["aadClientSecret"] = "123e4567-e89b-12d3-a456-426614174000"

					b, err := json.MarshalIndent(config, "", "\t")
					if err != nil {
						t.Error(err)
						t.FailNow()
					}
					cm.Data["config"] = string(b)
				}
				return cm
			},
			modify: func(cm *corev1.ConfigMap) {
				if _, ok := cm.Data["config"]; ok {
					var config map[string]interface{}
					err := yaml.Unmarshal([]byte(cm.Data["config"]), &config)
					if err != nil {
						t.Error(err)
						t.FailNow()
					}
					config["aadClientSecret"] = "123e4567-e89b-12d3-a456-426614174000"

					b, err := json.MarshalIndent(config, "", "\t")
					if err != nil {
						t.Error(err)
						t.FailNow()
					}
					cm.Data["config"] = string(b)
				}
			},
		},
	} {
		cm := tt.configMap()

		// modify expected result after fixup
		expected := getTestConfigMap()
		if tt.modify != nil {
			tt.modify(expected)
		}

		i := &Installer{
			kubernetescli: k8sfake.NewSimpleClientset(getTestSecret(), cm),
			log:           logrus.NewEntry(logrus.StandardLogger()),
		}
		err := i.fixCloudConfig(context.Background())
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		result, err := i.kubernetescli.CoreV1().ConfigMaps("openshift-config").Get("cloud-provider-config", metav1.GetOptions{})
		if err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(expected, result) {
			t.Errorf("want:\n %w \n got: \n %w", expected, result)
		}
	}
}
