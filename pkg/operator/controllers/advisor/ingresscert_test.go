package advisor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/go-test/deep"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
)

func TestDefaultIngressCertificate(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name              string
		aroCluster        *arov1alpha1.Cluster
		clusterVersion    *configv1.ClusterVersion
		ingressController *operatorv1.IngressController
		expectedState     operatorv1.OperatorCondition
	}{
		{
			name: "run: has default certificate",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			clusterVersion: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{
					ClusterID: "00000000-0000-0000-0000-000000000001",
				},
			},
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "openshift-ingress-operator",
				},
				Spec: operatorv1.IngressControllerSpec{
					DefaultCertificate: &corev1.LocalObjectReference{
						Name: "00000000-0000-0000-0000-000000000001-ingress",
					},
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultIngressCertificate,
				Status:  operatorv1.ConditionTrue,
				Reason:  "CheckDone",
				Message: "Default ingress certificate in use",
			},
		},
		{
			name: "run: does not have default certificate",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			clusterVersion: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{
					ClusterID: "00000000-0000-0000-0000-000000000001",
				},
			},
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "openshift-ingress-operator",
				},
				Spec: operatorv1.IngressControllerSpec{
					DefaultCertificate: &corev1.LocalObjectReference{
						Name: "fancy-custom-cert",
					},
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultIngressCertificate,
				Status:  operatorv1.ConditionFalse,
				Reason:  "CheckDone",
				Message: "Custom ingress certificate in use: fancy-custom-cert",
			},
		},
		{
			name: "fail: no ingress controller",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			clusterVersion: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Spec: configv1.ClusterVersionSpec{
					ClusterID: "00000000-0000-0000-0000-000000000001",
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultIngressCertificate,
				Status:  operatorv1.ConditionUnknown,
				Reason:  "CheckFailed",
				Message: `ingresscontrollers.operator.openshift.io "default" not found`,
			},
		},
		{
			name: "fail: no clusterversion",
			aroCluster: &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
			},
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "openshift-ingress-operator",
				},
				Spec: operatorv1.IngressControllerSpec{
					DefaultCertificate: &corev1.LocalObjectReference{
						Name: "fancy-custom-cert",
					},
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultIngressCertificate,
				Status:  operatorv1.ConditionUnknown,
				Reason:  "CheckFailed",
				Message: `clusterversions.config.openshift.io "version" not found`,
			},
		},
	} {
		arocli := arofake.NewSimpleClientset()
		configcli := configfake.NewSimpleClientset()
		operatorcli := operatorfake.NewSimpleClientset()

		if tt.aroCluster != nil {
			arocli = arofake.NewSimpleClientset(tt.aroCluster)
		}
		if tt.clusterVersion != nil {
			configcli = configfake.NewSimpleClientset(tt.clusterVersion)
		}
		if tt.ingressController != nil {
			operatorcli = operatorfake.NewSimpleClientset(tt.ingressController)
		}

		sp := &IngressCertificateChecker{
			arocli:      arocli,
			configcli:   configcli,
			operatorcli: operatorcli,
		}

		t.Run(tt.name, func(t *testing.T) {
			err := sp.Check(ctx)

			if err != nil {
				t.Error(err)
			}

			cluster, err := arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			conds := []operatorv1.OperatorCondition{}

			// nil out time
			for _, c := range cluster.Status.AdvisorConditions {
				c.LastTransitionTime = metav1.NewTime(time.Time{})
				conds = append(conds, c)
			}

			errs := deep.Equal(conds, []operatorv1.OperatorCondition{tt.expectedState})
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
