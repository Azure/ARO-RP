package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	mock_deploy "github.com/Azure/ARO-RP/pkg/util/mocks/operator/deploy"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestEnsureAROOperator(t *testing.T) {
	ctx := context.Background()

	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	for _, tt := range []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		mocks   func(*mock_deploy.MockOperator)
		wantErr string
	}{
		{
			name: "create/update success",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						ClusterProfile: api.ClusterProfile{
							Version: "4.8.18",
						},
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "1.2.3.4",
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					CreateOrUpdate(gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "create/update failure",
			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles: []api.IngressProfile{
							{
								Visibility: api.VisibilityPublic,
								Name:       "default",
							},
						},
						ProvisioningState: api.ProvisioningStateAdminUpdating,
						ClusterProfile: api.ClusterProfile{
							Version: "4.8.18",
						},
						NetworkProfile: api.NetworkProfile{
							APIServerPrivateEndpointIP: "1.2.3.4",
						},
					},
				},
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					CreateOrUpdate(gomock.Any()).
					Return(errors.New("Mock return: CreateFailed"))
			},

			wantErr: "Mock return: CreateFailed",
		},
		{
			name: "enriched data not available - skip",

			doc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(key),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: key,
					Properties: api.OpenShiftClusterProperties{
						IngressProfiles:   nil,
						ProvisioningState: api.ProvisioningStateAdminUpdating,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dep := mock_deploy.NewMockOperator(controller)
			if tt.mocks != nil {
				tt.mocks(dep)
			}

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,

				aroOperatorDeployer: dep,
			}

			err := m.ensureAROOperator(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestAroDeploymentReady(t *testing.T) {
	ctx := context.Background()

	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	for _, tt := range []struct {
		name           string
		kubernetescli  *fake.Clientset
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient)
		mocks          func(*mock_deploy.MockOperator)
		wantRes        bool
	}{
		{
			name: "create/update success",

			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPublic,
									Name:       "default",
								},
							},
							ProvisioningState: api.ProvisioningStateAdminUpdating,
							ClusterProfile: api.ClusterProfile{
								Version: "4.8.18",
							},
							NetworkProfile: api.NetworkProfile{
								APIServerPrivateEndpointIP: "1.2.3.4",
							},
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(dep *mock_deploy.MockOperator) {
				dep.EXPECT().
					IsReady(gomock.Any()).
					Return(true, nil)
			},
			kubernetescli: fake.NewSimpleClientset(),
			wantRes:       true,
		},
		{
			name: "enriched data not available - skip",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, dbClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				doc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID: key,
						Properties: api.OpenShiftClusterProperties{
							IngressProfiles:   nil,
							ProvisioningState: api.ProvisioningStateAdminUpdating,
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)
				doc.Dequeues = 1
				checker.AddOpenShiftClusterDocuments(doc)
			},
			kubernetescli: fake.NewSimpleClientset(),
			wantRes:       true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dep := mock_deploy.NewMockOperator(controller)
			if tt.mocks != nil {
				tt.mocks(dep)
			}

			dbOpenShiftClusters, dbClient := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, dbClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Dequeue(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				log:                 logrus.NewEntry(logrus.StandardLogger()),
				doc:                 doc,
				db:                  dbOpenShiftClusters,
				kubernetescli:       tt.kubernetescli,
				aroOperatorDeployer: dep,
			}

			ok, err := m.aroDeploymentReady(ctx)
			if err != nil || ok != tt.wantRes {
				t.Error(err)
			}

			for _, err = range checker.CheckOpenShiftClusters(dbClient) {
				t.Error(err)
			}
		})
	}
}
