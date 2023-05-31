package node

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestReconciler(t *testing.T) {
	transitionTime := metav1.Time{Time: time.Now()}
	defaultAvailable := utilconditions.ControllerDefaultAvailable(ControllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(ControllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(ControllerName)
	defaultConditions := []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded}

	tests := []struct {
		name            string
		nodeName        string
		nodeObject      *corev1.Node
		clusterNotFound bool
		featureFlag     bool
		wantErr         string
		startConditions []operatorv1.OperatorCondition
		wantConditions  []operatorv1.OperatorCondition
	}{
		{
			name:     "node is a master, don't touch it",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Labels: map[string]string{
						"node-role.kubernetes.io/master": "true",
					},
				},
			},
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "node doesn't exist",
			nodeName: "nonexistent-node",
			nodeObject: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
				},
			},
			featureFlag:     true,
			wantErr:         `nodes "nonexistent-node" not found`,
			startConditions: defaultConditions,
			wantConditions: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            `nodes "nonexistent-node" not found`,
				},
			},
		},
		{
			name:            "can't fetch cluster instance",
			nodeName:        "aro-fake-node-0",
			nodeObject:      &corev1.Node{},
			clusterNotFound: true,
			featureFlag:     true,
			wantErr:         `clusters.aro.openshift.io "cluster" not found`,
		},
		{
			name:        "feature flag is false, don't touch it",
			nodeName:    "aro-fake-node-0",
			nodeObject:  &corev1.Node{},
			featureFlag: false,
			wantErr:     "",
		},
		{
			name:     "isDraining false, annotation start time is blank",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationDrainStartTime: "",
					},
				},
			},
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "isDraining false, delete our annotation",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "aro-fake-node-0",
					Annotations: map[string]string{
						annotationDrainStartTime: "some-start-time",
					},
				},
			},
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "isDraining false, node is unschedulable=false",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "isDraining false, annotationDesiredConfig is blank",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "isDraining false, annotationCurrentConfig is blank",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "isDraining false, no conditions are met",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "isDraining false, current config matches desired",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions:  defaultConditions,
		},
		{
			name:     "isDraining true, set annotation",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeProgressing,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            `Draining node aro-fake-node-0`,
				},
				defaultDegraded,
			},
		},
		{
			name:     "isDraining true, degraded state, unable to drain",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeProgressing,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            `Draining node aro-fake-node-0`,
				},
				defaultDegraded,
			},
		},
		{
			name:     `node has nil annotations, return ""`,
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "aro-fake-node-0",
					Annotations: nil,
				},
			},
			featureFlag:     true,
			wantErr:         "",
			startConditions: defaultConditions,
			wantConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				defaultProgressing,
				defaultDegraded,
			},
		},
		{
			name:     "node is draining, deadline was exceeded, execute the drain",
			nodeName: "aro-fake-node-0",
			nodeObject: &corev1.Node{
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
			startConditions: []operatorv1.OperatorCondition{
				defaultAvailable,
				{
					Type:               ControllerName + "Controller" + operatorv1.OperatorStatusTypeProgressing,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: transitionTime,
					Message:            `Draining node aro-fake-node-0`,
				},
				defaultDegraded,
			},
			wantConditions: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientBuilder := ctrlfake.NewClientBuilder()
			if !tt.clusterNotFound {
				cluster := &arov1alpha1.Cluster{
					ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
					Spec: arov1alpha1.ClusterSpec{
						InfraID: "aro-fake",
						OperatorFlags: arov1alpha1.OperatorFlags{
							controllerEnabled: strconv.FormatBool(tt.featureFlag),
						},
					},
				}
				if len(tt.startConditions) > 0 {
					cluster.Status.Conditions = append(cluster.Status.Conditions, tt.startConditions...)
				}
				clientBuilder = clientBuilder.WithObjects(cluster)
			}

			if tt.nodeObject != nil {
				clientBuilder = clientBuilder.WithObjects(tt.nodeObject)
			}

			client := clientBuilder.Build()

			r := NewReconciler(logrus.NewEntry(logrus.StandardLogger()), client, fake.NewSimpleClientset(tt.nodeObject))

			request := ctrl.Request{}
			request.Name = tt.nodeName
			request.Namespace = ""
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			utilconditions.AssertControllerConditions(t, ctx, client, tt.wantConditions)
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
