package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	consolefake "github.com/openshift/client-go/console/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

var clusterOperator = &configv1.ClusterOperator{
	ObjectMeta: metav1.ObjectMeta{
		Name: "aro",
	},
}

var consoleNotification = &consolev1.ConsoleNotification{
	ObjectMeta: metav1.ObjectMeta{
		Name: "aro-sre-unsupported-condition",
	},
}

func TestUnsupportedCondition(t *testing.T) {
	for _, tt := range []struct {
		name                string
		role                string
		nodeItems           []corev1.Node
		clusterOperator     *configv1.ClusterOperator
		consoleNotification *consolev1.ConsoleNotification
		wantStatus          configv1.ConditionStatus
	}{
		{
			name: "The worker nodes amount is less then 3",
			role: "master",
			nodeItems: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node2",
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				},
			},
			clusterOperator:     clusterOperator,
			consoleNotification: consoleNotification,
			wantStatus:          configv1.ConditionFalse,
		},
		{
			name: "Worker node count is not less than 3",
			role: "master",
			nodeItems: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node2",
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node3",
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				},
			},
			clusterOperator:     clusterOperator,
			consoleNotification: consoleNotification,
			wantStatus:          configv1.ConditionTrue,
		},
	} {
		kubernetescli := fake.NewSimpleClientset()
		configcli := configfake.NewSimpleClientset()
		consolecli := consolefake.NewSimpleClientset()

		if tt.nodeItems != nil {
			kubernetescli = fake.NewSimpleClientset(&corev1.NodeList{Items: tt.nodeItems})
		}

		if tt.clusterOperator != nil {
			configcli = configfake.NewSimpleClientset(tt.clusterOperator)
		}

		if tt.consoleNotification != nil {
			consolecli = consolefake.NewSimpleClientset(tt.consoleNotification)
		}

		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			ucc := &UnsupportedConditionChecker{
				log:           utillog.GetLogger(),
				kubernetescli: kubernetescli,
				consolecli:    consolecli,
				configcli:     configcli,
				role:          tt.role,
			}

			err := ucc.Check(ctx)

			if err != nil {
				t.Error(err)
			}

			_, err = kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
			if err != nil {
				t.Error(err)
			}

			co, err := configcli.ConfigV1().ClusterOperators().Get(ctx, clusterOperatorName, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}

			if gotStatus := co.Status.Conditions[0].Status; gotStatus != tt.wantStatus {
				t.Errorf("Incorrect status, want: %v, got: %v", tt.wantStatus, gotStatus)
			}
		})
	}
}
