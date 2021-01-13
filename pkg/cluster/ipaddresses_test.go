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
	mock_dnsmanager "github.com/Azure/ARO-RP/pkg/util/mocks/dns"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestUpdateOrCreateRouterIP(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name          string
		kubernetescli *fake.Clientset
		mocks         func(*mock_dnsmanager.MockManager)
		wantErr       string
	}

	for _, tt := range []*test{
		{
			name: "create/update success",
			mocks: func(dm *mock_dnsmanager.MockManager) {
				dm.EXPECT().
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

			dm := mock_dnsmanager.NewMockManager(controller)
			if tt.mocks != nil {
				tt.mocks(dm)
			}

			key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1"

			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
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
			})
			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			clusterdoc, err := openShiftClustersDatabase.Dequeue(ctx)
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				kubernetescli: tt.kubernetescli,
				dns:           dm,
				db:            openShiftClustersDatabase,
				doc:           clusterdoc,
			}

			err = m.createOrUpdateRouterIP(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
