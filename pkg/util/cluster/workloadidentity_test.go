package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
)

func TestDetermineRequiredPlatformWorkloadIdentityScopes(t *testing.T) {
	ctx := context.Background()
	subscriptionID := "test-subscription-id"
	clusterName := "test-cluster"
	vnetResourceGroup := "test-vnet-rg"
	diskEncryptionSetID := "/subscriptions/test-subscription-id/resourceGroups/test-rg/providers/Microsoft.Compute/diskEncryptionSets/test-des"
	roleDefinitionID := "/providers/Microsoft.Authorization/roleDefinitions/test-role-guid"

	expectedMasterSubnet := "/subscriptions/test-subscription-id/resourceGroups/test-vnet-rg/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/test-cluster-master"
	expectedWorkerSubnet := "/subscriptions/test-subscription-id/resourceGroups/test-vnet-rg/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/test-cluster-worker"
	expectedVnet := "/subscriptions/test-subscription-id/resourceGroups/test-vnet-rg/providers/Microsoft.Network/virtualNetworks/dev-vnet"

	tests := []struct {
		name          string
		mockSetup     func(*mock_authorization.MockRoleDefinitionsClient)
		expectedScope []string
		wantErr       string
	}{
		{
			name: "subnet permissions - returns master and worker subnets",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/subnets/join/action",
									"Microsoft.Network/virtualNetworks/subnets/read",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedMasterSubnet, expectedWorkerSubnet},
		},
		{
			name: "vnet permissions - returns vnet",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/read",
									"Microsoft.Network/virtualNetworks/write",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedVnet},
		},
		{
			name: "DES permissions - returns disk encryption set",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Compute/diskEncryptionSets/read",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{diskEncryptionSetID},
		},
		{
			name: "mixed permissions in separate blocks - returns all unique scopes",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/subnets/join/action",
								},
							},
							{
								Actions: &[]string{
									"Microsoft.Compute/diskEncryptionSets/read",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedMasterSubnet, expectedWorkerSubnet, diskEncryptionSetID},
		},
		{
			name: "duplicate permissions - deduplicates scopes",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/subnets/join/action",
								},
							},
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/subnets/read",
								},
							},
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/subnets/write",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedMasterSubnet, expectedWorkerSubnet},
		},
		{
			name: "empty permissions - returns error",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{},
					},
				}, nil)
			},
			wantErr: "no scopes determined",
		},
		{
			name: "nil permissions - returns error",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: nil,
					},
				}, nil)
			},
			wantErr: "no scopes determined",
		},
		{
			name: "unmatched permissions - returns error",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Storage/storageAccounts/read",
									"Microsoft.Compute/virtualMachines/read",
								},
							},
						},
					},
				}, nil)
			},
			wantErr: "no scopes determined",
		},
		{
			name: "GetByID fails - returns error",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{}, errors.New("API error"))
			},
			wantErr: "failed to get role definition",
		},
		{
			name: "subnet permissions with wildcard - returns subnets",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/subnets/*",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedMasterSubnet, expectedWorkerSubnet},
		},
		{
			name: "nil actions - returns error",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: nil,
							},
						},
					},
				}, nil)
			},
			wantErr: "no scopes determined",
		},
		{
			name: "vnet and subnet permissions in separate blocks - returns both without duplicates",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/read",
								},
							},
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/subnets/join/action",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedVnet, expectedMasterSubnet, expectedWorkerSubnet},
		},
		{
			name: "vnet and subnet permissions in same block - returns both",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/read",
									"Microsoft.Network/virtualNetworks/subnets/join/action",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedVnet, expectedMasterSubnet, expectedWorkerSubnet}, // All scopes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRoleDefinitions := mock_authorization.NewMockRoleDefinitionsClient(controller)
			tt.mockSetup(mockRoleDefinitions)

			c := &Cluster{
				log: logrus.NewEntry(logrus.StandardLogger()),
				Config: &ClusterConfig{
					SubscriptionID: subscriptionID,
					ClusterName:    clusterName,
				},
				roledefinitions: mockRoleDefinitions,
			}

			scopes, err := c.determineRequiredPlatformWorkloadIdentityScopes(ctx, roleDefinitionID, vnetResourceGroup, diskEncryptionSetID)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Sort both slices for comparison since map iteration order is non-deterministic
			sort.Strings(scopes)
			expectedSorted := make([]string, len(tt.expectedScope))
			copy(expectedSorted, tt.expectedScope)
			sort.Strings(expectedSorted)

			if len(scopes) != len(expectedSorted) {
				t.Fatalf("expected %d scopes, got %d\nexpected: %v\ngot: %v", len(expectedSorted), len(scopes), expectedSorted, scopes)
			}

			for i := range scopes {
				if scopes[i] != expectedSorted[i] {
					t.Errorf("scope[%d]: expected %q, got %q", i, expectedSorted[i], scopes[i])
				}
			}

			// Verify no duplicates
			scopeSet := make(map[string]bool)
			for _, scope := range scopes {
				if scopeSet[scope] {
					t.Errorf("duplicate scope found: %q", scope)
				}
				scopeSet[scope] = true
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexAny(s, substr) >= 0)
}

func indexAny(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
