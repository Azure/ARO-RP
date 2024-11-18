package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestCreateOrUpdateDenyAssignment(t *testing.T) {
	ctx := context.Background()
	clusterRGName := "test-cluster"
	m := &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	for _, tt := range []struct {
		name  string
		doc   *api.OpenShiftClusterDocument
		mocks func(*mock_features.MockDeploymentsClient)
	}{
		{
			name: "needs create - ServicePrincipalProfile",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName),
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							SPObjectID: fakeClusterSPObjectId,
						},
					},
				},
			},
			mocks: func(client *mock_features.MockDeploymentsClient) {
				var parameters map[string]interface{}
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRGName, gomock.Any(), mgmtfeatures.Deployment{
					Properties: &mgmtfeatures.DeploymentProperties{
						Template: &arm.Template{
							Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
							ContentVersion: "1.0.0.0",
							Resources: []*arm.Resource{
								m.denyAssignment(),
							},
						},
						Parameters: parameters,
						Mode:       mgmtfeatures.Incremental,
					},
				}).Return(nil)
			},
		},
		{
			name: "needs create - ServicePrincipalProfile - missing ServicePrincipalProfile",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName),
						},
					},
				},
			},
			mocks: func(client *mock_features.MockDeploymentsClient) {},
		},
		{
			name: "needs create - ServicePrincipalProfile - missing SPObjectID",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName),
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{},
					},
				},
			},
			mocks: func(client *mock_features.MockDeploymentsClient) {},
		},
		{
			name: "needs create - PlatformWorkloadIdentityProfile",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"anything": {
									ObjectID:   "00000000-0000-0000-0000-000000000000",
									ClientID:   "11111111-1111-1111-1111-111111111111",
									ResourceID: "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/something/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name",
								},
							},
						},
					},
				},
			},
			mocks: func(client *mock_features.MockDeploymentsClient) {
				var parameters map[string]interface{}
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRGName, gomock.Any(), mgmtfeatures.Deployment{
					Properties: &mgmtfeatures.DeploymentProperties{
						Template: &arm.Template{
							Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
							ContentVersion: "1.0.0.0",
							Resources: []*arm.Resource{
								m.denyAssignment(),
							},
						},
						Parameters: parameters,
						Mode:       mgmtfeatures.Incremental,
					},
				}).Return(nil)
			},
		},
		{
			name: "needs create - PlatformWorkloadIdentityProfile - missing ObjectID",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName),
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"anything": {
									ClientID:   "11111111-1111-1111-1111-111111111111",
									ResourceID: "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/something/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name",
								},
							},
						},
					},
				},
			},
			mocks: func(client *mock_features.MockDeploymentsClient) {},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			deployments := mock_features.NewMockDeploymentsClient(controller)

			_env.EXPECT().FeatureIsSet(env.FeatureDisableDenyAssignments).AnyTimes().Return(false)

			m.doc = tt.doc

			if tt.mocks != nil {
				tt.mocks(deployments)
			}

			m.env = _env
			m.deployments = deployments

			err := m.createOrUpdateDenyAssignment(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
