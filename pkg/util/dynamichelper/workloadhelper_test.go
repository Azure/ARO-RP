package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestHashWorkloadConfigs(t *testing.T) {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "certificates",
			Namespace: "openshift-azure-logging",
		},
		Data: map[string][]byte{
			"stuff": []byte("9485958"),
		},
	}
	cm := &corev1.ConfigMap{
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mdsd",
			Namespace: "openshift-azure-logging",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "certificates",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "certificates",
								},
							},
						},
						{
							Name: "fluent-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "fluent-config",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "fluentbit-audit",
							Image: "fluentbitImage",
							VolumeMounts: []corev1.VolumeMount{
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
							VolumeMounts: []corev1.VolumeMount{
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

	err := hashWorkloadConfigs([]kruntime.Object{cm, sec, ds})
	if err != nil {
		t.Fatal(err)
	}

	expect := map[string]string{
		"checksum/configmap-fluent-config": "aad6b208b25ce1becb4b9b6f14fce290f4e1bafd287813decc1e773ac2ec9c4e",
		"checksum/secret-certificates":     "e6963d3f1943a7bf44ebfda9d0dc2c2c8f2295dbeed320c654f3489c2aed1344",
	}

	if !reflect.DeepEqual(expect, ds.Spec.Template.Annotations) {
		t.Error(ds.Spec.Template.Annotations)
	}
}
