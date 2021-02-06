package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	mock_dns "github.com/Azure/ARO-RP/pkg/util/mocks/dns"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestCreateOrUpdateRouterIPFromCluster(t *testing.T) {
	ctx := context.Background()

	const (
		key = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"
	)

	for _, tt := range []struct {
		name           string
		kubernetescli  *fake.Clientset
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient)
		mocks          func(*mock_dns.MockManager)
		wantErr        string
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
								},
							},
							ProvisioningState: api.ProvisioningStateCreating,
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(doc)

				doc.Dequeues = 1
				doc.OpenShiftCluster.Properties.IngressProfiles[0].IP = "1.2.3.4"
				checker.AddOpenShiftClusterDocuments(doc)
			},
			mocks: func(dns *mock_dns.MockManager) {
				dns.EXPECT().
					CreateOrUpdateRouter(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			kubernetescli: fake.NewSimpleClientset(&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "router-default",
					Namespace: "openshift-ingress",
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{{
							IP: "1.2.3.4",
						}},
					},
				},
			}),
		},
		{
			name: "create/update failed",
			kubernetescli: fake.NewSimpleClientset(&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "router-default",
					Namespace: "openshift-ingress",
				},
			}),
			wantErr: "routerIP not found",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dns := mock_dns.NewMockManager(controller)
			if tt.mocks != nil {
				tt.mocks(dns)
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
				doc:           doc,
				db:            dbOpenShiftClusters,
				dns:           dns,
				kubernetescli: tt.kubernetescli,
			}

			err = m.createOrUpdateRouterIPFromCluster(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			for _, err = range checker.CheckOpenShiftClusters(dbClient) {
				t.Error(err)
			}
		})
	}
}
