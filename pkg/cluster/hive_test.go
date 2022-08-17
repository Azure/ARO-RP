package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/hive"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	testdatabase "github.com/Azure/ARO-RP/test/database"
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

func TestHiveCreateNamespace(t *testing.T) {
	for _, tt := range []struct {
		testName              string
		existingNamespaceName string
		newNamespaceName      string
		clusterManagerMock    func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager
		expectedNamespaceName string
		wantErr               string
	}{
		{
			testName:              "doesn't return error if cluster manager is nil",
			existingNamespaceName: "",
			newNamespaceName:      "new-namespace",
			expectedNamespaceName: "",
		},
		{
			testName:              "creates namespace if it doesn't exist",
			existingNamespaceName: "",
			newNamespaceName:      "new-namespace",
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				namespaceToReturn := &corev1.Namespace{}
				namespaceToReturn.Name = namespaceName
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateNamespace(gomock.Any()).Return(namespaceToReturn, nil)
				return mockClusterManager
			},
			expectedNamespaceName: "new-namespace",
			wantErr:               "",
		},
		{
			testName:              "doesn't create namespace if it already exists",
			existingNamespaceName: "existing-namespace",
			newNamespaceName:      "new-namespace",
			expectedNamespaceName: "existing-namespace",
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateNamespace(gomock.Any()).Times(0)
				return mockClusterManager
			},
		},
		{
			testName:              "returns error if cluster manager returns error",
			existingNamespaceName: "",
			newNamespaceName:      "new-namespace",
			expectedNamespaceName: "",
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateNamespace(gomock.Any()).Return(nil, fmt.Errorf("cluster manager error"))
				return mockClusterManager
			},
			wantErr: "cluster manager error",
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := createManagerForTests(t, tt.existingNamespaceName)

			if tt.clusterManagerMock != nil {
				m.hiveClusterManager = tt.clusterManagerMock(controller, tt.newNamespaceName)
			}

			err := m.hiveCreateNamespace(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace != tt.expectedNamespaceName {
				t.Errorf("expected namespace to be %s, got %s",
					tt.expectedNamespaceName, m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace)
			}
		},
		)
	}
}

func createManagerForTests(t *testing.T, existingNamespaceName string) *manager {
	fakeDb, _ := testdatabase.NewFakeOpenShiftClusters()
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"

	doc := &api.OpenShiftClusterDocument{
		Key: strings.ToLower(key),
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: key,
			Properties: api.OpenShiftClusterProperties{
				HiveProfile: api.HiveProfile{
					Namespace: existingNamespaceName,
				},
			},
		},
	}

	fixture := testdatabase.NewFixture().WithOpenShiftClusters(fakeDb)
	fixture.AddOpenShiftClusterDocuments(doc)
	err := fixture.Create()
	if err != nil {
		t.Fatal(err)
	}

	m := &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
		db:  fakeDb,
		doc: doc,
	}
	return m
}

func TestHiveEnsureResources(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		testName           string
		clusterManagerMock func(mockCtrl *gomock.Controller, m *manager) *mock_hive.MockClusterManager
		wantErr            string
	}{
		{
			testName: "returns error if cluster manager returns error",
			clusterManagerMock: func(mockCtrl *gomock.Controller, m *manager) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateOrUpdate(ctx, m.subscriptionDoc, m.doc).Return(fmt.Errorf("cluster manager error"))
				return mockClusterManager
			},
			wantErr: "cluster manager error",
		},
		{
			testName: "does not return error if cluster manager is nil",
		},
		{
			testName: "calls cluster manager CreateOrUpdate with correct parameters",
			clusterManagerMock: func(mockCtrl *gomock.Controller, m *manager) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateOrUpdate(ctx, m.subscriptionDoc, m.doc).Return(nil)
				return mockClusterManager
			},
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := createManagerForTests(t, "")

			if tt.clusterManagerMock != nil {
				m.hiveClusterManager = tt.clusterManagerMock(controller, m)
			}

			err := m.hiveEnsureResources(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		},
		)
	}
}

func TestHiveDeleteResources(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		testName           string
		namespace          string
		clusterManagerMock func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager
		wantErr            string
	}{
		{
			testName: "doesn't return error if cluster manager is nil",
		},
		{
			testName:  "deletes namespace if it exists",
			namespace: "existing-namespace",
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().Delete(ctx, namespaceName).Return(nil)
				return mockClusterManager
			},
		},
		{
			testName: "doesn't attempt to delete namespace if it doesn't exist",
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().Delete(ctx, namespaceName).Times(0)
				return mockClusterManager
			},
		},
		{
			testName:  "returns error if cluster manager returns error",
			namespace: "existing-namespace",
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().Delete(ctx, namespaceName).Return(fmt.Errorf("cluster manager error"))
				return mockClusterManager
			},
			wantErr: "cluster manager error",
		},
	} {
		t.Run(tt.testName, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := createManagerForTests(t, tt.namespace)

			if tt.clusterManagerMock != nil {
				m.hiveClusterManager = tt.clusterManagerMock(controller, m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace)
			}

			err := m.hiveDeleteResources(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
