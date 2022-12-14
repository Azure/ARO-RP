package ingresscertificatechecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheck(t *testing.T) {
	ctx := context.Background()

	clusterVersion := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: configv1.ClusterID("fake-cluster-id"),
		},
	}

	for _, tt := range []struct {
		name              string
		ingressController *operatorv1.IngressController
		clusterVersion    *configv1.ClusterVersion
		wantErr           string
	}{
		{
			name:           "default certificate is set",
			clusterVersion: clusterVersion,
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "fake-cluster-id-ingress",
			}),
		},
		{
			name:           "unexpected certificate name",
			clusterVersion: clusterVersion,
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "fake-custom-name-ingress",
			}),
			wantErr: `custom ingress certificate in use: "fake-custom-name-ingress"`,
		},
		{
			name:              "no default certificate set",
			clusterVersion:    clusterVersion,
			ingressController: fakeIngressController(nil),
			wantErr:           "ingress has no default certificate set",
		},
		{
			name:           "missing IngressController",
			clusterVersion: clusterVersion,
			wantErr:        `ingresscontrollers.operator.openshift.io "default" not found`,
		},
		{
			name:    "missing ClusterVersion",
			wantErr: `clusterversions.config.openshift.io "version" not found`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			operatorcliFake := operatorfake.NewSimpleClientset()
			if tt.ingressController != nil {
				operatorcliFake.Tracker().Add(tt.ingressController)
			}
			configcliFake := configfake.NewSimpleClientset()
			if tt.clusterVersion != nil {
				configcliFake.Tracker().Add(tt.clusterVersion)
			}

			sp := &checker{
				operatorcli: operatorcliFake,
				configcli:   configcliFake,
			}

			err := sp.Check(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%s\n !=\n%s", err, tt.wantErr)
			}
		})
	}
}

func fakeIngressController(certificateRef *corev1.LocalObjectReference) *operatorv1.IngressController {
	return &operatorv1.IngressController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "openshift-ingress-operator",
		},
		Spec: operatorv1.IngressControllerSpec{
			DefaultCertificate: certificateRef,
		},
	}
}
