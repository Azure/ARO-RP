package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitOpenshiftApiServerStatuses(t *testing.T) {
	cli := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "apiserver",
				Namespace: "openshift-apiserver",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				NumberAvailable:        2,
			},
		},
	)

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		cli: cli,
		m:   m,
	}

	m.EXPECT().EmitGauge("apiserver.statuses", int64(1), map[string]string{
		"desired":   strconv.Itoa(3),
		"available": strconv.Itoa(2),
	})

	ctx := context.Background()
	err := mon.emitOpenshiftApiServerStatuses(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmitKubeApiServerStatuses(t *testing.T) {
	for _, tt := range []struct {
		name            string
		conditionType   string
		conditionStatus operatorv1.ConditionStatus
		expectEmit      bool
	}{
		{
			name:            "openshift-apiservers available",
			conditionType:   "Available",
			conditionStatus: operatorv1.ConditionTrue,
			expectEmit:      false,
		},
		{
			name:            "openshift-apiservers not available",
			conditionType:   "Available",
			conditionStatus: operatorv1.ConditionFalse,
			expectEmit:      true,
		},
		{
			name:            "Installer pod not degraded",
			conditionType:   "InstallerPodPendingDegraded",
			conditionStatus: operatorv1.ConditionFalse,
			expectEmit:      false,
		},
		{
			name:            "Installer pod is degraded",
			conditionType:   "InstallerPodPendingDegraded",
			conditionStatus: operatorv1.ConditionTrue,
			expectEmit:      true,
		},
	} {
		operatorcli := operatorfake.NewSimpleClientset(
			&operatorv1.KubeAPIServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: operatorv1.KubeAPIServerStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						OperatorStatus: operatorv1.OperatorStatus{
							Conditions: []operatorv1.OperatorCondition{
								{
									Type:   tt.conditionType,
									Status: tt.conditionStatus,
								},
							},
						},
					},
				},
			},
		)

		controller := gomock.NewController(t)
		defer controller.Finish()

		m := mock_metrics.NewMockInterface(controller)

		mon := &Monitor{
			operatorcli: operatorcli,
			m:           m,
		}

		if tt.expectEmit {
			m.EXPECT().EmitGauge("apiserver.conditions", int64(1), map[string]string{
				"type":   tt.conditionType,
				"status": string(tt.conditionStatus),
			})
		}

		ctx := context.Background()
		err := mon.emitKubeApiServerStatuses(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestEmitKubeApiServerNodeRevisionStatuses(t *testing.T) {
	for _, tt := range []struct {
		name         string
		nodeStatuses []operatorv1.NodeStatus
		expectEmit   bool
		indexEmitted int // the nodeStatuses' index that is emitted if expectEmit=True
	}{
		{
			name: "nodes at same revision",
			nodeStatuses: []operatorv1.NodeStatus{
				{
					NodeName:        "node1",
					CurrentRevision: 2,
					TargetRevision:  2,
				},
				{
					NodeName:        "node2",
					CurrentRevision: 2,
					TargetRevision:  2,
				},
			},
			expectEmit: false,
		},
		{
			name: "nodes at same target revision but different current revision",
			nodeStatuses: []operatorv1.NodeStatus{
				{
					NodeName:        "node1",
					CurrentRevision: 2,
					TargetRevision:  2,
				},
				{
					NodeName:        "node2",
					CurrentRevision: 1,
					TargetRevision:  2,
				},
			},
			expectEmit:   true,
			indexEmitted: 1,
		},
	} {
		operatorcli := operatorfake.NewSimpleClientset(
			&operatorv1.KubeAPIServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: operatorv1.KubeAPIServerStatus{
					StaticPodOperatorStatus: operatorv1.StaticPodOperatorStatus{
						NodeStatuses: tt.nodeStatuses,
					},
				},
			},
		)

		controller := gomock.NewController(t)
		defer controller.Finish()

		m := mock_metrics.NewMockInterface(controller)

		mon := &Monitor{
			operatorcli: operatorcli,
			m:           m,
		}

		if tt.expectEmit {
			s := tt.nodeStatuses[tt.indexEmitted]
			m.EXPECT().EmitGauge("apiserver.nodestatuses", int64(1), map[string]string{
				"name":    s.NodeName,
				"current": fmt.Sprintf("%d", s.CurrentRevision),
				"target":  fmt.Sprintf("%d", s.TargetRevision),
			})
		}

		ctx := context.Background()
		err := mon.emitKubeApiServerNodeRevisionStatuses(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
}
