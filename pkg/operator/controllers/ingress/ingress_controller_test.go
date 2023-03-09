package ingress

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

func TestReconciler(t *testing.T) {
	fakeCluster := func(controllerEnabledFlag string) *arov1alpha1.Cluster {
		return &arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: arov1alpha1.SingletonClusterName,
			},
			Spec: arov1alpha1.ClusterSpec{
				OperatorFlags: arov1alpha1.OperatorFlags{
					"aro.ingress.enabled": controllerEnabledFlag,
				},
			},
		}
	}

	tests := []struct {
		name                  string
		controllerEnabledFlag string
		ingressController     *operatorv1.IngressController
		expectedReplica       int32
		expectedError         string
	}{
		{
			name:                  "aro ingress controller disabled",
			controllerEnabledFlag: "false",
		},
		{
			name:                  "openshift ingress controller not found",
			controllerEnabledFlag: "true",
			expectedError:         "ingresscontrollers.operator.openshift.io \"default\" not found",
		},
		{
			name:                  "openshift ingress controller has 3 replicas",
			controllerEnabledFlag: "true",
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
			name:                  "openshift ingress controller has 2 replicas (minimum required replicas)",
			controllerEnabledFlag: "true",
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
			name:                  "openshift ingress controller has 1 replica",
			controllerEnabledFlag: "true",
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
			name:                  "openshift ingress controller has 0 replica",
			controllerEnabledFlag: "true",
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
			clusterMock := fakeCluster(tt.controllerEnabledFlag)

			clientBuilder := ctrlfake.NewClientBuilder().WithObjects(clusterMock)
			if tt.ingressController != nil {
				clientBuilder = clientBuilder.WithObjects(tt.ingressController)
			}
			clientFake := clientBuilder.Build()

			r := &Reconciler{
				log:    logrus.NewEntry(logrus.StandardLogger()),
				client: clientFake,
			}

			request := ctrl.Request{}
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)
			if err != nil && err.Error() != tt.expectedError {
				t.Error(err)
			}

			if tt.ingressController != nil {
				ingress := &operatorv1.IngressController{}
				err = r.client.Get(ctx, types.NamespacedName{Namespace: openshiftIngressControllerNamespace, Name: openshiftIngressControllerName}, ingress)
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
