package ingresscertificatechecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
)

type testData struct {
	name              string
	ingressController *operatorv1.IngressController
	clusterVersion    *configv1.ClusterVersion
	cluster           *arov1alpha1.Cluster
	wantErr           string
}

func TestCheck(t *testing.T) {
	const (
		managedDomain = "foo.location.aroapp.io"
		customDomain  = "custom.domain.io"
	)

	ctx := context.Background()

	testCases := []testData{
		{
			name:           "Check returns clusterVersion not found error when ClusterVersion is nil",
			clusterVersion: nil,
			wantErr:        `clusterversions.config.openshift.io "version" not found`,
		},
		{
			name:              "Check returns IngressController not found error when IngressController is nil",
			clusterVersion:    fakeClusterVersion(),
			ingressController: nil,
			wantErr:           `ingresscontrollers.operator.openshift.io "default" not found`,
		},
		{
			name:              "Check returns Cluster not found error when Cluster is nil",
			cluster:           nil,
			clusterVersion:    fakeClusterVersion(),
			ingressController: fakeIngressController(nil),
			wantErr:           "clusters.aro.openshift.io \"cluster\" not found",
		},
		{
			name:              "Check returns certificate error (ARO's fault) when no certificate is set and cluster has a managed domain",
			clusterVersion:    fakeClusterVersion(),
			ingressController: fakeIngressController(nil),
			cluster:           fakeCluster(managedDomain),
			wantErr:           errNoCertificateAndManagedDomain.Error(),
		},
		{
			name:              "Check returns error (customer's fault) when no certificate is set and cluster has a custom domain",
			clusterVersion:    fakeClusterVersion(),
			ingressController: fakeIngressController(nil),
			cluster:           fakeCluster(customDomain),
			wantErr:           errNoCertificateAndCustomDomain.Error(),
		},
		{
			name:           "Check returns error when cluster has a managed domain and there is an invalid certificate name",
			clusterVersion: fakeClusterVersion(),
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "fake-custom-name-ingress",
			}),
			cluster: fakeCluster(managedDomain),
			wantErr: errInvalidCertificateAndManagedDomain.Error(),
		},
		{
			name:           "Check returns no error when cluster has a managed domain and certificate has a valid name",
			clusterVersion: fakeClusterVersion(),
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "fake-cluster-id-ingress",
			}),
			cluster: fakeCluster(managedDomain),
		},
		{
			name:           "Check returns error when cluster has a managed domain and certificate name is empty",
			clusterVersion: fakeClusterVersion(),
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "",
			}),
			cluster: fakeCluster(managedDomain),
			wantErr: errInvalidCertificateAndManagedDomain.Error(),
		},
		{
			name:           "Check returns error when cluster has a managed domain and certificate name is just the ingress suffix",
			clusterVersion: fakeClusterVersion(),
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "-ingress",
			}),
			cluster: fakeCluster(managedDomain),
			wantErr: errInvalidCertificateAndManagedDomain.Error(),
		},
		{
			name:           "Check returns no error when we do not have a managed domain and certificate name is empty",
			clusterVersion: fakeClusterVersion(),
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "",
			}),
			cluster: fakeCluster(customDomain),
		},
		{
			name:           "Check returns no error when we do not have a managed domain and certificate name is just the suffix",
			clusterVersion: fakeClusterVersion(),
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "-ingress",
			}),
			cluster: fakeCluster(customDomain),
		},
		{
			name:           "Check returns no error when we do not have a managed domain and certificate name is invalid",
			clusterVersion: fakeClusterVersion(),
			ingressController: fakeIngressController(&corev1.LocalObjectReference{
				Name: "invalid-ingress-name",
			}),
			cluster: fakeCluster(customDomain),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sp := &checker{
				client: buildFakeClient(tt),
			}

			err := sp.Check(ctx)
			if err != nil && err.Error() != tt.wantErr || err == nil && tt.wantErr != "" {
				t.Errorf("\nExpected:\n%s \nbut got:\n%s", tt.wantErr, err)
			}
		})
	}
}

func buildFakeClient(testCase testData) client.WithWatch {
	clientBuilder := testclienthelper.NewAROFakeClientBuilder()
	if testCase.clusterVersion != nil {
		clientBuilder = clientBuilder.WithObjects(testCase.clusterVersion)
	}

	if testCase.ingressController != nil {
		clientBuilder = clientBuilder.WithObjects(testCase.ingressController)
	}

	if testCase.cluster != nil {
		clientBuilder = clientBuilder.WithObjects(testCase.cluster)
	}

	return clientBuilder.Build()
}

func fakeClusterVersion() *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: configv1.ClusterID("fake-cluster-id"),
		},
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

func fakeCluster(domain string) *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			Domain: domain,
		},
	}
}
