package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	hivev1alpha1 "github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitClusterSync(t *testing.T) {
	for _, tt := range []struct {
		name              string
		clusterSync       *hivev1alpha1.ClusterSync
		getClusterSyncErr error
		expectedError     error
		expectedGauges    []struct {
			name   string
			value  int64
			labels map[string]string
		}
	}{
		{
			name: "SyncSets and SelectorSyncSets have elements",
			clusterSync: &hivev1alpha1.ClusterSync{
				Status: hivev1alpha1.ClusterSyncStatus{
					SyncSets: []hivev1alpha1.SyncStatus{
						{
							Name:               "syncset1",
							Result:             "Success",
							FirstSuccessTime:   &metav1.Time{Time: time.Now()},
							LastTransitionTime: metav1.Time{Time: time.Now()},
							FailureMessage:     "",
						},
					},
					SelectorSyncSets: []hivev1alpha1.SyncStatus{
						{
							Name:               "selectorsyncset1",
							Result:             "Success",
							FirstSuccessTime:   &metav1.Time{Time: time.Now()},
							LastTransitionTime: metav1.Time{Time: time.Now()},
							FailureMessage:     "",
						},
					},
				},
			},
			expectedError: nil,
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{
				{
					name:  "hive.clustersync",
					value: 1,
					labels: map[string]string{
						"metric": "SyncSets",
						"name":   "syncset1",
						"result": "Success",
					},
				},
				{
					name:  "hive.clustersync",
					value: 1,
					labels: map[string]string{
						"metric": "SelectorSyncSets",
						"name":   "selectorsyncset1",
						"result": "Success",
					},
				},
			},
		},
		{
			name: "SyncSets and SelectorSyncSets have success and failure",
			clusterSync: &hivev1alpha1.ClusterSync{
				Status: hivev1alpha1.ClusterSyncStatus{
					SyncSets: []hivev1alpha1.SyncStatus{
						{
							Name:               "syncset2",
							Result:             "Failure",
							FirstSuccessTime:   &metav1.Time{Time: time.Now()},
							LastTransitionTime: metav1.Time{Time: time.Now()},
							FailureMessage:     "",
						},
					},
					SelectorSyncSets: []hivev1alpha1.SyncStatus{
						{
							Name:               "selectorsyncset2",
							Result:             "Success",
							FirstSuccessTime:   &metav1.Time{Time: time.Now()},
							LastTransitionTime: metav1.Time{Time: time.Now()},
							FailureMessage:     "",
						},
					},
				},
			},
			expectedError: nil,
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{
				{
					name:  "hive.clustersync",
					value: 1,
					labels: map[string]string{
						"metric": "SyncSets",
						"name":   "syncset2",
						"result": "Failure",
					},
				},
				{
					name:  "hive.clustersync",
					value: 1,
					labels: map[string]string{
						"metric": "SelectorSyncSets",
						"name":   "selectorsyncset2",
						"result": "Success",
					},
				},
			},
		},
		{
			name: "SyncSets and SelectorSyncSets are nil",
			clusterSync: &hivev1alpha1.ClusterSync{
				Status: hivev1alpha1.ClusterSyncStatus{
					SyncSets:         nil,
					SelectorSyncSets: nil,
				},
			},
			expectedError: nil,
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{},
		},
		{
			name:              "GetSyncSetResources returns error",
			getClusterSyncErr: errors.New("some error"),
			expectedError:     errors.New("some error"),
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHiveClusterManager := mock_hive.NewMockClusterManager(ctrl)
			m := mock_metrics.NewMockEmitter(ctrl)

			mockMonitor := &Monitor{
				hiveClusterManager: mockHiveClusterManager,
				m:                  m,
			}

			ctx := context.Background()

			mockHiveClusterManager.EXPECT().GetClusterSync(ctx, mockMonitor.doc).Return(tt.clusterSync, tt.getClusterSyncErr).AnyTimes()

			for _, gauge := range tt.expectedGauges {
				m.EXPECT().EmitGauge(gauge.name, gauge.value, gauge.labels).Times(1)
			}

			err := mockMonitor.emitClusterSync(ctx)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}
