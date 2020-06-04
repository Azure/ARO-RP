package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestHashWorkloadConfigs(t *testing.T) {
	sec := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "certificates",
			Namespace: "openshift-azure-logging",
		},
		StringData: map[string]string{
			"stuff": "9485958",
		},
	}
	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluent-config",
			Namespace: "openshift-azure-logging",
		},
		Data: map[string]string{
			"audit.conf":      "auditConf",
			"containers.conf": "containersConf",
			"journal.conf":    "journalConf",
			"parsers.conf":    "parsersConf",
		},
	}
	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mdsd",
			Namespace: "openshift-azure-logging",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "certificates",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "certificates",
								},
							},
						},
						{
							Name: "fluent-config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "fluent-config",
									},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "fluentbit-audit",
							Image: "fluentbitImage",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "fluent-config",
									ReadOnly:  true,
									MountPath: "/etc/td-agent-bit",
								},
							},
						},
						{
							Name:  "mdsd",
							Image: "mdsdImage",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "certificates",
									MountPath: "/etc/mdsd.d/secret",
								},
							},
						},
					},
				},
			},
		},
	}

	var objects []*unstructured.Unstructured
	for _, res := range []runtime.Object{cm, sec, ds} {
		un, err := ToUnstructured(res)
		if err != nil {
			t.Error(err)
		}
		objects = append(objects, un)
	}
	HashWorkloadConfigs(objects)
	expect := map[string]string{
		"checksum/configmap-fluent-config": "290a2fb8ebdfcff1a489f434b2ac527dfe9af9ce94aadb6151f024f75272972b",
		"checksum/secret-certificates":     "4829fa88cccbf7344f31ada2499e5a25ae906f99b9c6d1e4a271a5e71abe6cce",
	}
	newAnnotations, ok, err := unstructured.NestedStringMap(objects[2].Object, "spec", "template", "metadata", "annotations")
	if err != nil || !ok {
		t.Error(err)
		t.Error(ok)
	}
	if !reflect.DeepEqual(expect, newAnnotations) {
		t.Error(newAnnotations)
	}
}
