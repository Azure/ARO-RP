package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"

	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
)

func TestDetermineRequiredPlatformWorkloadIdentityScopes(t *testing.T) {
	ctx := context.Background()
	vnetResourceGroup := "test-vnet-rg"
	roleDefinitionID := "/providers/Microsoft.Authorization/roleDefinitions/test-role-guid"
	diskEncryptionSetID := "/subscriptions/test-subscription-id/resourceGroups/test-rg/providers/Microsoft.Compute/diskEncryptionSets/test-des"
	expectedRouteTable := "/subscriptions/test-subscription-id/resourceGroups/test-vnet-rg/providers/Microsoft.Network/routeTables/test-cluster-rt"

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
			name: "route table permissions - returns route table",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/routeTables/read",
									"Microsoft.Network/routeTables/write",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedRouteTable},
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
			name: "vnet, route table, and DES permissions - returns all",
			mockSetup: func(m *mock_authorization.MockRoleDefinitionsClient) {
				m.EXPECT().GetByID(ctx, roleDefinitionID).Return(mgmtauthorization.RoleDefinition{
					RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
						Permissions: &[]mgmtauthorization.Permission{
							{
								Actions: &[]string{
									"Microsoft.Network/virtualNetworks/write",
									"Microsoft.Network/routeTables/join/action",
									"Microsoft.Compute/diskEncryptionSets/read",
								},
							},
						},
					},
				}, nil)
			},
			expectedScope: []string{expectedVnet, expectedRouteTable, diskEncryptionSetID},
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
			name: "vnet and subnet permissions in separate blocks - returns vnet only (inheritance)",
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
			expectedScope: []string{expectedVnet},
		},
		{
			name: "vnet and subnet permissions in same block - returns vnet only (inheritance)",
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
			expectedScope: []string{expectedVnet},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRoleDefinitionsClient := mock_authorization.NewMockRoleDefinitionsClient(controller)
			tt.mockSetup(mockRoleDefinitionsClient)

			c := &Cluster{
				log: logrus.NewEntry(logrus.StandardLogger()),
				Config: &ClusterConfig{
					SubscriptionID: "test-subscription-id",
					ClusterName:    "test-cluster",
				},
				roledefinitions: mockRoleDefinitionsClient,
			}

			scopes, err := c.determineRequiredPlatformWorkloadIdentityScopes(ctx, roleDefinitionID, vnetResourceGroup, diskEncryptionSetID, expectedRouteTable)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
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

func TestCreateRoleAssignmentWithRetry(t *testing.T) {
	ctx := context.Background()
	scope := "/subscriptions/test-sub/resourceGroups/test-rg"
	roleDefinitionID := "/providers/Microsoft.Authorization/roleDefinitions/test-role"
	principalID := "test-principal-id"

	tests := []struct {
		name      string
		mockSetup func(*mock_authorization.MockRoleAssignmentsClient)
		wantErr   bool
	}{
		{
			name: "success on first attempt",
			mockSetup: func(m *mock_authorization.MockRoleAssignmentsClient) {
				m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, nil)
			},
			wantErr: false,
		},
		{
			name: "HTTP 409 Conflict - role assignment already exists (idempotent)",
			mockSetup: func(m *mock_authorization.MockRoleAssignmentsClient) {
				conflictErr := autorest.DetailedError{
					StatusCode: http.StatusConflict,
				}
				m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, conflictErr)
			},
			wantErr: false,
		},
		{
			name: "transient error then success on retry",
			mockSetup: func(m *mock_authorization.MockRoleAssignmentsClient) {
				// First call fails with transient error
				gomock.InOrder(
					m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, errors.New("HashConflictOnDifferentRoleAssignmentIds")),
					// Second call succeeds
					m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, nil),
				)
			},
			wantErr: false,
		},
		{
			name: "permanent failure after max retries",
			mockSetup: func(m *mock_authorization.MockRoleAssignmentsClient) {
				// All attempts fail
				m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, errors.New("permanent error")).Times(roleAssignmentMaxRetries)
			},
			wantErr: true,
		},
		{
			name: "multiple transient errors then success",
			mockSetup: func(m *mock_authorization.MockRoleAssignmentsClient) {
				gomock.InOrder(
					m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, errors.New("transient error 1")),
					m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, errors.New("transient error 2")),
					m.EXPECT().Create(gomock.Any(), scope, gomock.Any(), gomock.Any()).Return(mgmtauthorization.RoleAssignment{}, nil),
				)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockRoleAssignmentsClient := mock_authorization.NewMockRoleAssignmentsClient(controller)
			tt.mockSetup(mockRoleAssignmentsClient)

			c := &Cluster{
				log:             logrus.NewEntry(logrus.StandardLogger()),
				roleassignments: mockRoleAssignmentsClient,
				retryDelay:      0, // Skip sleeps in tests
			}

			err := c.createRoleAssignmentWithRetry(ctx, scope, roleDefinitionID, principalID)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
