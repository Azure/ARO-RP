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
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestReconciler(t *testing.T) {
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

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
		startConditions       []operatorv1.OperatorCondition
		wantConditions        []operatorv1.OperatorCondition
	}{
		{
			name:                  "aro ingress controller disabled",
			controllerEnabledFlag: "false",
			startConditions:       defaultConditions,
			wantConditions:        defaultConditions,
		},
		{
			name:                  "openshift ingress controller not found",
			controllerEnabledFlag: "true",
			expectedError:         "ingresscontrollers.operator.openshift.io \"default\" not found",
			startConditions:       defaultConditions,
			wantConditions:        defaultConditions,
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
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
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
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
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
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
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
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
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

			r := NewReconciler(logrus.NewEntry(logrus.StandardLogger()), clientFake)

			request := ctrl.Request{}
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)
			utilerror.AssertErrorMessage(t, err, tt.expectedError)

			if tt.ingressController != nil {
				ingress := &operatorv1.IngressController{}
				err = r.Client.Get(ctx, types.NamespacedName{Namespace: openshiftIngressControllerNamespace, Name: openshiftIngressControllerName}, ingress)
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
