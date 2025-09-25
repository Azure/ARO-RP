package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hiveinternal/v1alpha1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type expectedMetric struct {
	name   string
	value  any
	labels map[string]string
}

func TestMonitor(t *testing.T) {
	ctx := context.Background()

	innerFailure := errors.New("failure inside")
	fakeNamespace := "foobar"

	for _, tt := range []struct {
		name           string
		expectedErrors []error
		mocks          func(f *mock_hive.MockClusterManager)
		collectors     func(*Monitor) []collectorFunc
		expectedGauges []expectedMetric
	}{
		{
			name: "happy path",
			collectors: func(m *Monitor) []collectorFunc {
				return []collectorFunc{m.emitHiveRegistrationStatus}
			},
			mocks: func(f *mock_hive.MockClusterManager) {
				f.EXPECT().GetClusterDeployment(gomock.Any(), gomock.Any()).Return(&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      hive.ClusterDeploymentName,
						Namespace: fakeNamespace,
					},
					Status: hivev1.ClusterDeploymentStatus{
						Conditions: []hivev1.ClusterDeploymentCondition{
							{ //should be returned
								Type:   hivev1.ClusterReadyCondition,
								Status: corev1.ConditionFalse,
							},
						},
					},
				}, nil)
			},
			expectedGauges: []expectedMetric{
				{
					name:  "hive.clusterdeployment.conditions",
					value: int64(1),
					labels: map[string]string{
						"reason": "",
						"type":   "Ready",
					},
				},
			},
		},
		{
			name: "collector failure",
			collectors: func(m *Monitor) []collectorFunc {
				return []collectorFunc{m.emitHiveRegistrationStatus}
			},
			mocks: func(f *mock_hive.MockClusterManager) {
				f.EXPECT().GetClusterDeployment(gomock.Any(), gomock.Any()).Return(nil, innerFailure)
			},
			expectedErrors: []error{
				&failureToRunHiveCollector{collectorName: "emitHiveRegistrationStatus"},
				innerFailure,
			},
			expectedGauges: []expectedMetric{
				{
					name:  "monitor.hive.collector.error",
					value: int64(1),
					labels: map[string]string{
						"collector": "emitHiveRegistrationStatus",
					},
				},
			},
		},
		{
			name: "collector panic does not stop other collectors",
			collectors: func(m *Monitor) []collectorFunc {
				return []collectorFunc{m.emitClusterSync, m.emitHiveRegistrationStatus}
			},
			mocks: func(f *mock_hive.MockClusterManager) {
				f.EXPECT().GetClusterDeployment(gomock.Any(), gomock.Any()).Return(&hivev1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      hive.ClusterDeploymentName,
						Namespace: fakeNamespace,
					},
					Status: hivev1.ClusterDeploymentStatus{
						Conditions: []hivev1.ClusterDeploymentCondition{
							{ //should be returned
								Type:   hivev1.ClusterReadyCondition,
								Status: corev1.ConditionFalse,
							},
						},
					},
				}, nil)
				f.EXPECT().GetClusterSync(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, oc *api.OpenShiftCluster) (*v1alpha1.ClusterSync, error) {
					panic(innerFailure)
				})
			},
			expectedErrors: []error{
				&failureToRunHiveCollector{collectorName: "emitClusterSync"},
				&collectorPanic{panicValue: innerFailure},
			},
			expectedGauges: []expectedMetric{
				{
					name:  "hive.clusterdeployment.conditions",
					value: int64(1),
					labels: map[string]string{
						"reason": "",
						"type":   "Ready",
					},
				},
				{
					name:  "monitor.hive.collector.error",
					value: int64(1),
					labels: map[string]string{
						"collector": "emitClusterSync",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.New()
			controller := gomock.NewController(t)
			m := mock_metrics.NewMockEmitter(controller)

			fakeHive := mock_hive.NewMockClusterManager(controller)

			if tt.mocks != nil {
				tt.mocks(fakeHive)
			}

			oc := &api.OpenShiftCluster{
				Name: "testcluster",
				Properties: api.OpenShiftClusterProperties{
					HiveProfile: api.HiveProfile{
						Namespace: fakeNamespace,
					},
				},
			}

			mon := &Monitor{
				oc:                 oc,
				log:                log,
				m:                  m,
				hiveClusterManager: fakeHive,
			}

			if tt.collectors != nil {
				mon.collectors = tt.collectors(mon)
			}

			for _, gauge := range tt.expectedGauges {
				m.EXPECT().EmitGauge(gauge.name, gauge.value, gauge.labels).Times(1)
			}

			// we only emit duration when no errors
			if len(tt.expectedErrors) == 0 {
				m.EXPECT().EmitFloat("monitor.hive.duration", gomock.Any(), gomock.Any()).Times(1)
			}

			err := mon.Monitor(ctx)
			utilerror.AssertErrorMatchesAll(t, err, tt.expectedErrors)
		})
	}
}
