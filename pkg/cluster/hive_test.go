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
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestHiveClusterDeploymentReady(t *testing.T) {
	fakeNamespace := "aro-00000000-0000-0000-0000-000000000000"
	for _, tt := range []struct {
		name       string
		mocks      func(hiveMock *mock_hive.MockClusterManager, doc *api.OpenShiftClusterDocument)
		wantResult bool
		wantErr    string
	}{
		{
			name: "ready",
			mocks: func(hiveMock *mock_hive.MockClusterManager, doc *api.OpenShiftClusterDocument) {
				hiveMock.EXPECT().IsClusterDeploymentReady(gomock.Any(), doc).Return(true, nil)
			},
			wantResult: true,
		},
		{
			name: "not ready",
			mocks: func(hiveMock *mock_hive.MockClusterManager, doc *api.OpenShiftClusterDocument) {
				hiveMock.EXPECT().IsClusterDeploymentReady(gomock.Any(), doc).Return(false, nil)
			},
			wantResult: false,
		},
		{
			name: "error",
			mocks: func(hiveMock *mock_hive.MockClusterManager, doc *api.OpenShiftClusterDocument) {
				hiveMock.EXPECT().IsClusterDeploymentReady(gomock.Any(), doc).Return(false, errors.New("fake err"))
			},
			wantResult: false,
			wantErr:    "fake err",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := createManagerForTests(t, fakeNamespace)

			hiveMock := mock_hive.NewMockClusterManager(controller)
			if tt.mocks != nil {
				tt.mocks(hiveMock, m.doc)
			}
			m.hiveClusterManager = hiveMock

			result, _, err := m.hiveClusterDeploymentReady(context.Background())
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantResult != result {
				t.Error(result)
			}
		})
	}
}

func TestHiveResetCorrelationData(t *testing.T) {
	fakeNamespace := "aro-00000000-0000-0000-0000-000000000000"

	for _, tt := range []struct {
		name  string
		mocks func(hiveMock *mock_hive.MockClusterManager, doc *api.OpenShiftClusterDocument)

		wantErr string
	}{
		{
			name: "success",
			mocks: func(hiveMock *mock_hive.MockClusterManager, doc *api.OpenShiftClusterDocument) {
				hiveMock.EXPECT().ResetCorrelationData(gomock.Any(), doc).Return(nil)
			},
		},
		{
			name: "error",
			mocks: func(hiveMock *mock_hive.MockClusterManager, doc *api.OpenShiftClusterDocument) {
				hiveMock.EXPECT().ResetCorrelationData(gomock.Any(), doc).Return(errors.New("fake err"))
			},
			wantErr: "fake err",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			m := createManagerForTests(t, fakeNamespace)

			hiveMock := mock_hive.NewMockClusterManager(controller)
			if tt.mocks != nil {
				tt.mocks(hiveMock, m.doc)
			}
			m.hiveClusterManager = hiveMock

			err := m.hiveResetCorrelationData(context.Background())
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestHiveCreateNamespace(t *testing.T) {
	fakeNamespace := "aro-00000000-0000-0000-0000-000000000000"
	fakeNewNamespace := "aro-11111111-1111-1111-1111-111111111111"
	for _, tt := range []struct {
		testName              string
		existingNamespaceName string
		newNamespaceName      string
		clusterManagerMock    func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager
		expectedNamespaceName string
		wantErr               string
	}{
		{
			testName:              "creates namespace if it doesn't exist",
			existingNamespaceName: "",
			newNamespaceName:      fakeNamespace,
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				namespaceToReturn := &corev1.Namespace{}
				namespaceToReturn.Name = namespaceName
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateNamespace(gomock.Any(), gomock.Any()).Return(namespaceToReturn, nil)
				return mockClusterManager
			},
			expectedNamespaceName: fakeNamespace,
			wantErr:               "",
		},
		{
			testName:              "doesn't create namespace if it already exists",
			existingNamespaceName: fakeNamespace,
			newNamespaceName:      fakeNewNamespace,
			expectedNamespaceName: fakeNamespace,
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateNamespace(gomock.Any(), gomock.Any()).Times(0)
				return mockClusterManager
			},
		},
		{
			testName:              "returns error if cluster manager returns error",
			existingNamespaceName: "",
			newNamespaceName:      fakeNamespace,
			expectedNamespaceName: "",
			clusterManagerMock: func(mockCtrl *gomock.Controller, namespaceName string) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().CreateNamespace(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("cluster manager error"))
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
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

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

	return &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
		db:  fakeDb,
		doc: doc,

		adoptViaHive:   true,
		installViaHive: true,
	}
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
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		},
		)
	}
}

func TestHiveDeleteResources(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		testName           string
		namespace          string
		clusterManagerMock func(mockCtrl *gomock.Controller, doc *api.OpenShiftClusterDocument) *mock_hive.MockClusterManager
		wantErr            string
	}{
		{
			testName:  "deletes namespace if it exists",
			namespace: "existing-namespace",
			clusterManagerMock: func(mockCtrl *gomock.Controller, doc *api.OpenShiftClusterDocument) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().Delete(ctx, doc).Return(nil)
				return mockClusterManager
			},
		},
		{
			testName: "doesn't attempt to delete namespace if it doesn't exist",
			clusterManagerMock: func(mockCtrl *gomock.Controller, doc *api.OpenShiftClusterDocument) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().Delete(ctx, doc).Times(0)
				return mockClusterManager
			},
		},
		{
			testName:  "returns error if cluster manager returns error",
			namespace: "existing-namespace",
			clusterManagerMock: func(mockCtrl *gomock.Controller, doc *api.OpenShiftClusterDocument) *mock_hive.MockClusterManager {
				mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
				mockClusterManager.EXPECT().Delete(ctx, doc).Return(fmt.Errorf("cluster manager error"))
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
				m.hiveClusterManager = tt.clusterManagerMock(controller, m.doc)
			}

			err := m.hiveDeleteResources(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
