package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitSummary(t *testing.T) {
	tests := []struct {
		name             string
		cvs              []runtime.Object
		nodes            []runtime.Object
		oc               *api.OpenShiftCluster
		nodeListReaction ktesting.ReactionFunc
		wantDims         map[string]string
	}{
		{
			name: "no errors",
			cvs: []runtime.Object{
				&configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Status: configv1.ClusterVersionStatus{
						Desired: configv1.Update{
							Version: "4.3.3",
						},
						History: []configv1.UpdateHistory{
							{
								State:   configv1.CompletedUpdate,
								Version: "4.3.0",
							},
						},
					},
				},
			},
			nodes: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-0",
						Labels: map[string]string{
							masterRoleLabel: "",
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-node-1",
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-node-2",
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:       api.ProvisioningStateFailed,
					FailedProvisioningState: api.ProvisioningStateDeleting,
				},
			},
			wantDims: map[string]string{
				"actualVersion":           "4.3.0",
				"desiredVersion":          "4.3.3",
				"masterCount":             "1",
				"workerCount":             "2",
				"provisioningState":       api.ProvisioningStateFailed.String(),
				"failedProvisioningState": api.ProvisioningStateDeleting.String(),
			},
		},
		{
			name: "error getting cluster version",
			nodes: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-master-0",
						Labels: map[string]string{
							masterRoleLabel: "",
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-node-1",
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aro-node-2",
						Labels: map[string]string{
							workerRoleLabel: "",
						},
					},
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:       api.ProvisioningStateFailed,
					FailedProvisioningState: api.ProvisioningStateDeleting,
				},
			},
			wantDims: map[string]string{
				"actualVersion":           "unknown",
				"desiredVersion":          "unknown",
				"masterCount":             "1",
				"workerCount":             "2",
				"provisioningState":       api.ProvisioningStateFailed.String(),
				"failedProvisioningState": api.ProvisioningStateDeleting.String(),
			},
		},
		{
			name: "error getting nodes",
			cvs: []runtime.Object{
				&configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Status: configv1.ClusterVersionStatus{
						Desired: configv1.Update{
							Version: "4.3.3",
						},
						History: []configv1.UpdateHistory{
							{
								State:   configv1.CompletedUpdate,
								Version: "4.3.0",
							},
						},
					},
				},
			},
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState:       api.ProvisioningStateFailed,
					FailedProvisioningState: api.ProvisioningStateDeleting,
				},
			},
			nodeListReaction: func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, errors.New("fake error")
			},
			wantDims: map[string]string{
				"actualVersion":           "4.3.0",
				"desiredVersion":          "4.3.3",
				"masterCount":             "unknown",
				"workerCount":             "unknown",
				"provisioningState":       api.ProvisioningStateFailed.String(),
				"failedProvisioningState": api.ProvisioningStateDeleting.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockInterface(controller)

			cli := fake.NewSimpleClientset(tt.nodes...)
			if tt.nodeListReaction != nil {
				cli.PrependReactor("list", "nodes", tt.nodeListReaction)
			}

			mon := &Monitor{
				log:       logrus.NewEntry(logrus.StandardLogger()),
				configcli: configfake.NewSimpleClientset(tt.cvs...),
				cli:       cli,
				m:         m,
				oc:        tt.oc,
				hourlyRun: true,
			}

			m.EXPECT().EmitGauge("cluster.summary", int64(1), tt.wantDims)

			err := mon.emitSummary(context.Background())
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
