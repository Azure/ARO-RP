package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
)

func TestFixNSG(t *testing.T) {
	ctx := context.Background()
	subscriptionID := "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"

	tests := []struct {
		name       string
		infraID    string
		visibility api.Visibility
		mocks      func(*mock_network.MockSecurityGroupsClient)
		wantErr    string
	}{
		{
			name:       "private/good",
			infraID:    "test",
			visibility: api.VisibilityPrivate,
			mocks: func(nsgc *mock_network.MockSecurityGroupsClient) {
				nsgc.EXPECT().Get(gomock.Any(), "test-cluster", "test-controlplane-nsg", "").Return(
					mgmtnetwork.SecurityGroup{}, nil)
			},
		},
		{
			name:       "private/needs fix",
			infraID:    "test",
			visibility: api.VisibilityPrivate,
			mocks: func(nsgc *mock_network.MockSecurityGroupsClient) {
				nsgc.EXPECT().Get(gomock.Any(), "test-cluster", "test-controlplane-nsg", "").Return(
					mgmtnetwork.SecurityGroup{
						SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{
							SecurityRules: &[]mgmtnetwork.SecurityRule{
								{
									SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
										Protocol:             mgmtnetwork.SecurityRuleProtocolTCP,
										DestinationPortRange: to.StringPtr("6443"),
									},
								},
							},
						},
					}, nil)

				nsgc.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "test-cluster", "test-controlplane-nsg",
					mgmtnetwork.SecurityGroup{
						SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{
							SecurityRules: &[]mgmtnetwork.SecurityRule{},
						},
					}).Return(nil)
			},
		},
		{
			name:       "public/good",
			visibility: api.VisibilityPublic,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			securitygroupsClient := mock_network.NewMockSecurityGroupsClient(controller)
			if tt.mocks != nil {
				tt.mocks(securitygroupsClient)
			}

			i := &Installer{
				securitygroups: securitygroupsClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							InfraID: tt.infraID,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", subscriptionID),
							},
							APIServerProfile: api.APIServerProfile{
								Visibility: tt.visibility,
							},
						},
					},
				},
			}

			err := i.fixNSG(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
