package cluster

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//Test that namespace is created if it doesn't exist
func TestHiveCreateNamespaceShouldCreateNamespace(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	namespaceNameToSet := "namespace-to-set"
	existingNamespaceName := ""

	m, err := createManagerForTests(t, existingNamespaceName)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	namespaceToReturn := &v1.Namespace{}
	namespaceToReturn.Name = namespaceNameToSet
	mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
	mockClusterManager.EXPECT().CreateNamespace(ctx).Return(namespaceToReturn, nil)

	m.hiveClusterManager = mockClusterManager

	err = m.hiveCreateNamespace(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	//assert that namespace is set
	if m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace != namespaceNameToSet {
		t.Errorf("expected namespace to be %s, got %s",
			namespaceNameToSet, m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace)
	}
}

func TestHiveCreateNamespaceShouldNotReturnErrorIfClusterManagerIsNil(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	existingNamespaceName := ""

	m, err := createManagerForTests(t, existingNamespaceName)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	m.hiveClusterManager = nil

	//assert that no error occurs
	err = m.hiveCreateNamespace(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	//assert that namespace is not updated
	if m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace != existingNamespaceName {
		t.Errorf("expected namespace to remain %s, got %s",
			existingNamespaceName, m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace)
	}
}

func TestHiveCreateNamespaceShouldNotCreateNamespaceIfItAlreadyExists(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	existingNamespaceName := "existing-namespace"

	m, err := createManagerForTests(t, existingNamespaceName)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
	mockClusterManager.EXPECT().CreateNamespace(gomock.Any()).Times(0)

	m.hiveClusterManager = mockClusterManager

	err = m.hiveCreateNamespace(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	//assert that namespace is not updated
	if m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace != existingNamespaceName {
		t.Errorf("expected namespace to remain %s, got %s",
			existingNamespaceName, m.doc.OpenShiftCluster.Properties.HiveProfile.Namespace)
	}
}

func TestHiveCreateNamespaceShouldReturnErrorIfNamespaceCreationFails(t *testing.T) {
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m, err := createManagerForTests(t, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	mockClusterManager := mock_hive.NewMockClusterManager(mockCtrl)
	mockClusterManager.EXPECT().CreateNamespace(ctx).Return(nil, fmt.Errorf("error"))

	m.hiveClusterManager = mockClusterManager

	err = m.hiveCreateNamespace(ctx)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func createManagerForTests(t *testing.T, existingNamespaceName string) (*manager, error) {
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
	return m, err
}
