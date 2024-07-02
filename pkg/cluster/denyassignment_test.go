package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

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
				Identity: &api.Identity{
					UserAssignedIdentities: api.UserAssignedIdentities{
						"fakeIdentity": api.ClusterUserAssignedIdentity{
							ClientID:    "fake",
							PrincipalID: "alsoFake",
						},
					},
				},
			},
		},
	}

	for _, tt := range []struct {
		name  string
		mocks func(*mock_features.MockDeploymentsClient)
	}{
		{
			name: "needs create",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			deployments := mock_features.NewMockDeploymentsClient(controller)

			_env.EXPECT().FeatureIsSet(env.FeatureDisableDenyAssignments).AnyTimes().Return(false)

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
