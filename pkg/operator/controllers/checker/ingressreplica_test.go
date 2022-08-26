package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
)

func TestIngressReplicaChecker(t *testing.T) {
	ctx := context.Background()

	var defaultReplica int32 = 2
	var zeroReplica int32 = 0

	for _, tt := range []struct {
		name              string
		aroCluster        *arov1alpha1.Cluster
		ingressController *operatorv1.IngressController
		expectedState     operatorv1.OperatorCondition
	}{
		{
			name: "ingress controller has default replicas",
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
					Replicas: &defaultReplica,
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultIngressReplica,
				Status:  operatorv1.ConditionTrue,
				Message: "Default replicas in place",
				Reason:  "CheckDone",
			},
		},
		{
			name: "ingress controller has zero replicas",
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
					Replicas: &zeroReplica,
				},
			},
			expectedState: operatorv1.OperatorCondition{
				Type:    arov1alpha1.DefaultIngressReplica,
				Status:  operatorv1.ConditionTrue,
				Message: "Rescale succeeded",
				Reason:  "CheckDone",
			},
		},
	} {
		arocli := arofake.NewSimpleClientset()
		operatorcli := operatorfake.NewSimpleClientset()

		if tt.aroCluster != nil {
			arocli = arofake.NewSimpleClientset(tt.aroCluster)
		}

		if tt.ingressController != nil {
			operatorcli = operatorfake.NewSimpleClientset(tt.ingressController)
		}

		replicaChecker := NewIngressReplicaChecker(arocli, operatorcli, "")

		t.Run(tt.name, func(t *testing.T) {
			err := replicaChecker.Check(ctx)

			if err != nil {
				t.Error(err)
			}

			cluster, err := arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if cluster.Status.Conditions[0].Message != tt.expectedState.Message {
				t.Error("wrong condition message")
			}
			if cluster.Status.Conditions[0].Reason != tt.expectedState.Reason {
				t.Error("wrong condition reason")
			}
			if cluster.Status.Conditions[0].Type != tt.expectedState.Type {
				t.Error("wrong condition type")
			}
			if cluster.Status.Conditions[0].LastTransitionTime.IsZero() {
				t.Error("zero last transition time")
			}
		})
	}
}
