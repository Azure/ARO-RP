package node

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestIsDraining(t *testing.T) {
	for _, tt := range []struct {
		name string
		node *corev1.Node
		want bool
	}{
		{
			name: "current config doesn't match desired",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "bar",
					},
				},
			},
			want: false,
		},
		{
			name: "node is not ready",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						},
					},
				}},
			want: false,
		},
		{
			name: "node is unschedulable=false",
			node: &corev1.Node{
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
			want: false,
		},
		{
			name: "current config matches desired config",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "foo",
					},
				},
			},
			want: false,
		},
		{
			name: "annotationDesiredConfig is blank",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "",
					},
				},
			},
			want: false,
		},
		{
			name: "annotationCurrentConfig is blank",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         "noop",
						annotationReason:        "no reason",
						annotationCurrentConfig: "",
						annotationDesiredConfig: "foo",
					},
				},
			},
			want: false,
		},
		{
			name: "state is working",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         stateWorking,
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "bar",
					},
				},
			},
			want: true,
		},
		{
			name: "state is degraded and annotationReason is correct",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         stateDegraded,
						annotationReason:        "failed to drain node",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "bar",
					},
				},
			},
			want: true,
		},
		{
			name: "state is degraded but annotationReason is incorrect",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         stateDegraded,
						annotationReason:        "some-random-reason",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "bar",
					},
				},
			},
			want: false,
		},
		{
			name: "annotationReason is correct but state is incorrect",
			node: &corev1.Node{
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
					Annotations: map[string]string{
						annotationState:         "not-a-valid-state",
						annotationReason:        "failed to drain node",
						annotationCurrentConfig: "foo",
						annotationDesiredConfig: "bar",
					},
				},
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDraining(tt.node); tt.want != got {
				t.Error(got)
			}
		})
	}
}

func TestGetAnnotation(t *testing.T) {
	for _, tt := range []struct {
		name            string
		objectMeta      *metav1.ObjectMeta
		annotationKey   string
		annotationValue string
	}{
		{
			name:            "no annotations set, return a blank annotation",
			objectMeta:      &metav1.ObjectMeta{},
			annotationValue: "",
		},
		{
			name: "annotation is set, return its value",
			objectMeta: &metav1.ObjectMeta{
				Annotations: map[string]string{"some-random-annotation": "is-set"},
			},
			annotationKey:   "some-random-annotation",
			annotationValue: "is-set",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAnnotation(tt.objectMeta, tt.annotationKey); tt.annotationValue != got {
				t.Error(got)
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
			name:            "ensure annotations are being set",
			node:            &corev1.Node{},
			annotationKey:   "foo",
			annotationValue: "bar",
		},
		{
			name: "ensure annotations are being overidden when new value is provided",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationCurrentConfig: "old",
					},
				},
			},
			annotationKey:   "machineconfiguration.openshift.io/currentConfig",
			annotationValue: "new",
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
