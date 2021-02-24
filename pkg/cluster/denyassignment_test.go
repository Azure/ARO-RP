package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	mock_authz "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestCreateOrUpdateDenyAssignment(t *testing.T) {
	ctx := context.Background()
	clusterRGName := "test-cluster"
	m := &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
		doc: &api.OpenShiftClusterDocument{
			OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterRGName),
					},
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						SPObjectID: fakeClusterSPObjectId,
					},
				},
			},
		},
		subscriptionDoc: &api.SubscriptionDocument{
			Subscription: &api.Subscription{
				Properties: &api.SubscriptionProperties{},
			},
		},
	}

	for _, tt := range []struct {
		name            string
		denyAssignments []mgmtauthorization.DenyAssignment
		mocks           func(*mock_features.MockDeploymentsClient)
	}{
		{

			name: "noop",
			denyAssignments: []mgmtauthorization.DenyAssignment{
				{
					DenyAssignmentProperties: &mgmtauthorization.DenyAssignmentProperties{
						ExcludePrincipals: &[]mgmtauthorization.Principal{
							{
								ID: to.StringPtr(fakeClusterSPObjectId),
							},
						},
					},
				},
			},
		},
		{

			name: "needs create",
			denyAssignments: []mgmtauthorization.DenyAssignment{
				{
					DenyAssignmentProperties: &mgmtauthorization.DenyAssignmentProperties{
						ExcludePrincipals: &[]mgmtauthorization.Principal{
							{
								ID: to.StringPtr("00000000-0000-0000-0000-000000000001"),
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
							Resources:      []*arm.Resource{m.denyAssignment()},
						},
						Parameters: parameters,
						Mode:       mgmtfeatures.Incremental,
					},
				}).Return(nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			denyAssignments := mock_authz.NewMockDenyAssignmentClient(controller)
			deployments := mock_features.NewMockDeploymentsClient(controller)

			env.EXPECT().DeploymentMode().Return(deployment.Production)
			denyAssignments.EXPECT().ListForResourceGroup(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.denyAssignments, nil)

			if tt.mocks != nil {
				tt.mocks(deployments)
			}

			m.env = env
			m.denyAssignments = denyAssignments
			m.deployments = deployments

			err := m.createOrUpdateDenyAssignment(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
