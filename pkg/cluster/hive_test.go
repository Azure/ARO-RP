package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
)

func TestHiveClusterDeploymentReady(t *testing.T) {
	fakeNamespace := "fake-namespace"

	for _, tt := range []struct {
		name  string
		mocks func(hiveMock *mock_hive.MockClusterManager)
		// TODO(hive): remove wantSkip and all test cases with wantSkip=True once we have Hive everywhere
		wantSkip   bool
		wantResult bool
		wantErr    string
	}{
		{
			name: "ready",
			mocks: func(hiveMock *mock_hive.MockClusterManager) {
				hiveMock.EXPECT().IsClusterDeploymentReady(gomock.Any(), fakeNamespace).Return(true, nil)
			},
			wantResult: true,
		},
		{
			name: "not ready",
			mocks: func(hiveMock *mock_hive.MockClusterManager) {
				hiveMock.EXPECT().IsClusterDeploymentReady(gomock.Any(), fakeNamespace).Return(false, nil)
			},
			wantResult: false,
		},
		{
			name: "error",
			mocks: func(hiveMock *mock_hive.MockClusterManager) {
				hiveMock.EXPECT().IsClusterDeploymentReady(gomock.Any(), fakeNamespace).Return(false, errors.New("fake err"))
			},
			wantResult: false,
			wantErr:    "fake err",
		},
		{
			name:       "skip",
			wantSkip:   true,
			wantResult: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			var hiveClusterManager hive.ClusterManager
			if !tt.wantSkip {
				hiveMock := mock_hive.NewMockClusterManager(controller)
				hiveClusterManager = hiveMock
				if tt.mocks != nil {
					tt.mocks(hiveMock)
				}
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							HiveProfile: api.HiveProfile{
								Namespace: fakeNamespace,
							},
						},
					},
				},
				hiveClusterManager: hiveClusterManager,
			}

			result, err := m.hiveClusterDeploymentReady(context.Background())
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if tt.wantResult != result {
				t.Error(result)
			}
		})
	}
}

func TestHiveResetCorrelationData(t *testing.T) {
	fakeNamespace := "fake-namespace"

	for _, tt := range []struct {
		name  string
		mocks func(hiveMock *mock_hive.MockClusterManager)
		// TODO(hive): remove wantSkip and all test cases with wantSkip=True once we have Hive everywhere
		wantSkip bool
		wantErr  string
	}{
		{
			name: "success",
			mocks: func(hiveMock *mock_hive.MockClusterManager) {
				hiveMock.EXPECT().ResetCorrelationData(gomock.Any(), fakeNamespace).Return(nil)
			},
		},
		{
			name: "error",
			mocks: func(hiveMock *mock_hive.MockClusterManager) {
				hiveMock.EXPECT().ResetCorrelationData(gomock.Any(), fakeNamespace).Return(errors.New("fake err"))
			},
			wantErr: "fake err",
		},
		{
			name:     "skip",
			wantSkip: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			var hiveClusterManager hive.ClusterManager
			if !tt.wantSkip {
				hiveMock := mock_hive.NewMockClusterManager(controller)
				hiveClusterManager = hiveMock
				if tt.mocks != nil {
					tt.mocks(hiveMock)
				}
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							HiveProfile: api.HiveProfile{
								Namespace: fakeNamespace,
							},
						},
					},
				},
				hiveClusterManager: hiveClusterManager,
			}

			err := m.hiveResetCorrelationData(context.Background())
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
