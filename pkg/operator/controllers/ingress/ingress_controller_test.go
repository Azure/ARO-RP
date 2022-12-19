package ingress

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestReconciler(t *testing.T) {
	fakeCluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			OperatorFlags: arov1alpha1.OperatorFlags{},
		},
	}
	tests := []struct {
		name                     string
		aroCluster               *arov1alpha1.Cluster
		aroIngressControllerFlag string
		ingressController        *operatorv1.IngressController
		expectedReplica          int32
		expectedError            string
	}{
		{
			name:                     "aro ingress controller disabled",
			aroCluster:               fakeCluster,
			aroIngressControllerFlag: "false",
		},
		{
			name:                     "openshift ingress controller not found",
			aroCluster:               fakeCluster,
			aroIngressControllerFlag: "true",
			expectedError:            "ingresscontrollers.operator.openshift.io \"default\" not found",
		},
		{
			name:                     "openshift ingress controller has 3 replicas",
			aroCluster:               fakeCluster,
			aroIngressControllerFlag: "true",
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      openshiftIngressControllerName,
					Namespace: openshiftIngressControllerNamespace,
				},
				Spec: operatorv1.IngressControllerSpec{
					Replicas: to.Int32Ptr(3),
				},
			},
			expectedReplica: 3,
		},
		{
			name:                     "openshift ingress controller has 2 replicas (minimum required replicas)",
			aroCluster:               fakeCluster,
			aroIngressControllerFlag: "true",
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      openshiftIngressControllerName,
					Namespace: openshiftIngressControllerNamespace,
				},
				Spec: operatorv1.IngressControllerSpec{
					Replicas: to.Int32Ptr(minimumReplicas),
				},
			},
			expectedReplica: minimumReplicas,
		},
		{
			name:                     "openshift ingress controller has 1 replica",
			aroCluster:               fakeCluster,
			aroIngressControllerFlag: "true",
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      openshiftIngressControllerName,
					Namespace: openshiftIngressControllerNamespace,
				},
				Spec: operatorv1.IngressControllerSpec{
					Replicas: to.Int32Ptr(1),
				},
			},
			expectedReplica: minimumReplicas,
		},
		{
			name:                     "openshift ingress controller has 0 replica",
			aroCluster:               fakeCluster,
			aroIngressControllerFlag: "true",
			ingressController: &operatorv1.IngressController{
				ObjectMeta: metav1.ObjectMeta{
					Name:      openshiftIngressControllerName,
					Namespace: openshiftIngressControllerNamespace,
				},
				Spec: operatorv1.IngressControllerSpec{
					Replicas: to.Int32Ptr(0),
				},
			},
			expectedReplica: minimumReplicas,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.aroCluster.Spec.OperatorFlags["aro.ingress.enabled"] = tt.aroIngressControllerFlag
			operatorcli := operatorfake.NewSimpleClientset()

			if tt.ingressController != nil {
				operatorcli = operatorfake.NewSimpleClientset(tt.ingressController)
			}

			r := &Reconciler{
				log:         logrus.NewEntry(logrus.StandardLogger()),
				operatorcli: operatorcli,
				client:      ctrlfake.NewClientBuilder().WithObjects(tt.aroCluster).Build(),
			}

			request := ctrl.Request{}
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)
			if err != nil && err.Error() != tt.expectedError {
				t.Error(err)
			}

			if tt.ingressController != nil {
				ingress, err := operatorcli.OperatorV1().IngressControllers(openshiftIngressControllerNamespace).Get(ctx, openshiftIngressControllerName, metav1.GetOptions{})
				if err != nil {
					t.Error(err)
				}
				if *ingress.Spec.Replicas != tt.expectedReplica {
					t.Errorf("incorrect replica count, expect: %d, got: %d", tt.expectedReplica, *ingress.Spec.Replicas)
				}
			}
		})
	}
}
