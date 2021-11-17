package node

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	ctrl "sigs.k8s.io/controller-runtime"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestReconciler(t *testing.T) {
	tests := []struct {
		name            string
		nodeName        string
		nodeObject      corev1.Node
		clusterNotFound bool
		featureFlag     bool
		wantErr         string
	}{
		{
			name:     "node is a master, don't touch it",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Labels: map[string]string{
						"node-role.kubernetes.io/master": "true",
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "node doesn't exist",
			nodeName: "nonexistent-node",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
				},
			},
			featureFlag: true,
			wantErr:     `nodes "nonexistent-node" not found`,
		},
		{
			name:            "can't fetch cluster instance",
			nodeName:        "aro-fake-node-0",
			nodeObject:      corev1.Node{},
			clusterNotFound: true,
			featureFlag:     true,
			wantErr:         `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name:        "feature flag is false, don't touch it",
			nodeName:    "aro-fake-node-0",
			nodeObject:  corev1.Node{},
			featureFlag: false,
			wantErr:     "",
		},
		{
			name:     "isDraining false, annotation start time is blank",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationDrainStartTime: "",
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining false, delete our annotation",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationDrainStartTime: "some-start-time",
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining false, node is unschedulable=false",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: false,
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining false, annotationDesiredConfig is blank",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "",
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining false, annotationCurrentConfig is blank",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "",
						annotationDesiredConfig: "foo",
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining false, no conditions are met",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "bar",
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining false, current config matches desired",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "foo",
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining true, set annotation",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationCurrentConfig: "config",
						annotationDesiredConfig: "config-2",
						annotationState:         stateWorking,
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining true, set annotation",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationCurrentConfig: "config",
						annotationDesiredConfig: "config-2",
						annotationState:         stateWorking,
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "isDraining true, degraded state, unable to drain",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationCurrentConfig: "config",
						annotationDesiredConfig: "config-2",
						annotationState:         stateDegraded,
						annotationReason:        "failed to drain node",
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     `node has nil annotations, return ""`,
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "aro-fake-node-0",
					Annotations: nil,
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
		{
			name:     "node is draining, deadline was exceeded, execute the drain",
			nodeName: "aro-fake-node-0",
			nodeObject: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationCurrentConfig:  "config",
						annotationDesiredConfig:  "config-2",
						annotationState:          stateDegraded,
						annotationReason:         "failed to drain node",
						annotationDrainStartTime: "2006-01-02T15:04:05Z",
					},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true,
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			featureFlag: true,
			wantErr:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseCluster := arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Spec: arov1alpha1.ClusterSpec{
					InfraID: "aro-fake",
					OperatorFlags: arov1alpha1.OperatorFlags{
						ENABLED: strconv.FormatBool(tt.featureFlag),
					},
				},
			}

			if tt.clusterNotFound == true {
				baseCluster = arov1alpha1.Cluster{}
			}

			r := &Reconciler{
				log: logrus.NewEntry(logrus.StandardLogger()),

				arocli:        arofake.NewSimpleClientset(&baseCluster),
				kubernetescli: fake.NewSimpleClientset(&tt.nodeObject),
			}

			request := ctrl.Request{}
			request.Name = tt.nodeName
			request.Namespace = ""
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestSetAnnotation(t *testing.T) {
	for _, tt := range []struct {
		name            string
		node            *corev1.Node
		annotationKey   string
		annotationValue string
	}{
		{
			// This test case is still necessary for coverage, because Reconcile only uses setAnnotaion() to set annotationDrainStartTime and never sets nil k/v.
			name: "ensure an empty map is returned when k and v are nil",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			annotationKey:   "",
			annotationValue: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setAnnotation(&tt.node.ObjectMeta, tt.annotationKey, tt.annotationValue)

			if !reflect.DeepEqual(tt.node.ObjectMeta.Annotations, map[string]string{tt.annotationKey: tt.annotationValue}) {
				t.Error(cmp.Diff(tt.node.ObjectMeta.Annotations, map[string]string{tt.annotationKey: tt.annotationValue}))
			}

		})
	}
}
