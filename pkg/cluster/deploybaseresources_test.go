package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	mgmtpolicy "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-09-01/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	utilrand "k8s.io/apimachinery/pkg/util/rand"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_authorization "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/authorization"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_policy "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/policy"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEnsureResourceGroup(t *testing.T) {
	ctx := context.Background()
	clusterID := "test-cluster"
	resourceGroupName := "fakeResourceGroup"
	resourceGroup := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", resourceGroupName)
	location := "eastus"

	group := mgmtfeatures.ResourceGroup{
		Location:  &location,
		ManagedBy: &clusterID,
	}

	groupWithTags := group
	groupWithTags.Tags = map[string]*string{
		"yeet": to.StringPtr("yote"),
	}

	resourceGroupManagedByMismatch := autorest.NewErrorWithError(&azure.RequestError{
		ServiceError: &azure.ServiceError{Code: "ResourceGroupManagedByMismatch"},
	}, "", "", nil, "")

	disallowedByPolicy := autorest.NewErrorWithError(&azure.RequestError{
		ServiceError: &azure.ServiceError{Code: "RequestDisallowedByPolicy"},
	}, "", "", nil, "")

	for _, tt := range []struct {
		name              string
		provisioningState api.ProvisioningState
		mocks             func(*mock_features.MockResourceGroupsClient, *mock_env.MockInterface)
		modify            func(*manager)
		wantErr           string
	}{
		{
			name:              "success - rg doesn't exist",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, group).
					Return(group, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), gomock.Any(), resourceGroupName).
					Return(nil)
			},
		},
		{
			name:              "success - rg doesn't exist and localdev mode tags set",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				groupWithLocalDevTags := group
				groupWithLocalDevTags.Tags = map[string]*string{
					"purge": to.StringPtr("true"),
				}
				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, groupWithLocalDevTags).
					Return(groupWithLocalDevTags, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(true)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), gomock.Any(), resourceGroupName).
					Return(nil)
			},
		},
		{
			name:              "success - rg doesn't exist and customer's tags set",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				groupWithCustomerTags := group
				groupWithCustomerTags.Tags = map[string]*string{
					"foo": to.StringPtr("bar"),
					"bar": to.StringPtr("baz"),
				}

				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, groupWithCustomerTags).
					Return(groupWithCustomerTags, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), gomock.Any(), resourceGroupName).
					Return(nil)
			},
			modify: modifyManagerResourceTags,
		},
		{
			name:              "success - rg doesn't exist and customer's tags and localdev mode tags both set",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				groupWithMixedTags := group
				groupWithMixedTags.Tags = map[string]*string{
					"purge": to.StringPtr("true"),
					"foo":   to.StringPtr("bar"),
					"bar":   to.StringPtr("baz"),
				}

				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, groupWithMixedTags).
					Return(groupWithMixedTags, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(true)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), gomock.Any(), resourceGroupName).
					Return(nil)
			},
			modify: modifyManagerResourceTags,
		},
		{
			name:              "success - rg exists and maintain tags",
			provisioningState: api.ProvisioningStateAdminUpdating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					Get(gomock.Any(), resourceGroupName).
					Return(groupWithTags, nil)

				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, groupWithTags).
					Return(groupWithTags, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), gomock.Any(), resourceGroupName).
					Return(nil)
			},
		},
		{
			name:              "success - rg exists, maintains existing tags, and is updated correctly with new tags",
			provisioningState: api.ProvisioningStateAdminUpdating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				groupWithAddedTags := groupWithTags
				groupWithAddedTags.Tags["foo"] = to.StringPtr("bar")
				groupWithAddedTags.Tags["bar"] = to.StringPtr("baz")

				rg.EXPECT().
					Get(gomock.Any(), resourceGroupName).
					Return(groupWithTags, nil)

				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, groupWithAddedTags).
					Return(groupWithAddedTags, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), gomock.Any(), resourceGroupName).
					Return(nil)
			},
			modify: modifyManagerResourceTags,
		},
		{
			name:              "fail - get rg returns generic error",
			provisioningState: api.ProvisioningStateAdminUpdating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					Get(gomock.Any(), resourceGroupName).
					Return(group, errors.New("generic error"))
			},
			wantErr: "generic error",
		},
		{
			name:              "fail - CreateOrUpdate returns resourcegroupmanagedbymismatch",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, group).
					Return(group, resourceGroupManagedByMismatch)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)
			},
			wantErr: "400: ClusterResourceGroupAlreadyExists: : Resource group " + resourceGroup + " must not already exist.",
		},
		{
			name:              "fail - CreateOrUpdate returns requestdisallowedbypolicy",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, group).
					Return(group, disallowedByPolicy)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)
			},
			wantErr: `400: DeploymentFailed: : Deployment failed. Details: : : {"code":"RequestDisallowedByPolicy","message":"","target":null,"details":null,"innererror":null,"additionalInfo":null}`,
		},
		{
			name:              "fail - CreateOrUpdate returns generic error",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, group).
					Return(group, errors.New("generic error"))

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)
			},
			wantErr: "generic error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			resourceGroupsClient := mock_features.NewMockResourceGroupsClient(controller)
			env := mock_env.NewMockInterface(controller)
			tt.mocks(resourceGroupsClient, env)

			env.EXPECT().Location().AnyTimes().Return(location)

			m := &manager{
				log:            logrus.NewEntry(logrus.StandardLogger()),
				resourceGroups: resourceGroupsClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroup,
							},
							ProvisioningState: tt.provisioningState,
						},
						Location: location,
						ID:       clusterID,
					},
				},
				env: env,
			}

			if tt.modify != nil {
				tt.modify(m)
			}

			err := m.ensureResourceGroup(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEnsureTaggingPolicy(t *testing.T) {
	ctx := context.Background()
	location := "eastus"
	infraID := "arouser-xx2nx"
	uuid := "123e4567-e89b-12d3-a456-426614174000"

	resourceGroupName := "fakeResourceGroup"
	resourceGroup := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", resourceGroupName)

	displayName := tagPolicyDisplayName(infraID)

	definitionID := "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/00000000-0000-0000-0000-000000000000"
	spID := "00000000-0000-0000-0000-000000000000"

	// This would be painfully long if fully typed out here.
	// E2E tests will validate the policy's functional correctness.
	definition := resourceTaggingPolicyDefinition(displayName)

	definitionWithID := definition
	definitionWithID.ID = to.StringPtr(definitionID)

	assignment := mgmtpolicy.Assignment{
		Location: to.StringPtr(location),
		Identity: &mgmtpolicy.Identity{
			Type: mgmtpolicy.SystemAssigned,
		},
		AssignmentProperties: &mgmtpolicy.AssignmentProperties{
			DisplayName:        to.StringPtr(displayName),
			PolicyDefinitionID: to.StringPtr(definitionID),
			Scope:              to.StringPtr(resourceGroup),
			EnforcementMode:    mgmtpolicy.Default,
		},
	}

	assignmentWithSpID := mgmtpolicy.Assignment{
		Identity: &mgmtpolicy.Identity{
			PrincipalID: to.StringPtr(spID),
		},
	}

	roleAssignmentParams := mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/providers/Microsoft.Authorization/roleDefinitions/4a9ae827-6dc8-4573-8ac7-8239d42aa03f"), // Tag Contributor
			PrincipalID:      to.StringPtr(spID),
			PrincipalType:    mgmtauthorization.ServicePrincipal,
		},
	}

	tagSet := map[string]string{
		"foo":  "foo",
		"bar":  "bar",
		"bash": "bash",
		"zsh":  "zsh",
	}

	tagSetKeys := []string{}

	for k := range tagSet {
		tagSetKeys = append(tagSetKeys, k)
	}

	sort.Strings(tagSetKeys)

	maxLenTagSet := map[string]string{}
	maxLenTagSetKeys := []string{}

	for i := 0; i < maxTags; i++ {
		k := fmt.Sprintf("key%d", i)
		maxLenTagSet[k] = fmt.Sprintf("value%d", i)
		maxLenTagSetKeys = append(maxLenTagSetKeys, k)
	}

	sort.Strings(maxLenTagSetKeys)

	for _, tt := range []struct {
		name                 string
		modify               func(*manager)
		assignmentsMocks     func(*mock_policy.MockAssignmentsClient)
		definitionsMocks     func(*mock_policy.MockDefinitionsClient)
		roleAssignmentsMocks func(*mock_authorization.MockRoleAssignmentsClient)
		wantErr              string
	}{
		{
			name: "empty ResourceTags in cluster doc - policy parameters all empty strings",
			definitionsMocks: func(definitions *mock_policy.MockDefinitionsClient) {
				definitionWithID := definition
				definitionWithID.ID = to.StringPtr(definitionID)
				definitions.EXPECT().CreateOrUpdate(ctx, displayName, definition).Return(definitionWithID, nil)
			},
			assignmentsMocks: func(assignments *mock_policy.MockAssignmentsClient) {
				assignmentWithParams := assignment
				parameters := map[string]*mgmtpolicy.ParameterValuesValue{}

				for i := 0; i < maxTags; i++ {
					tagKeyParamName := tagKeyParamName(i)
					tagValueParamName := tagValueParamName(i)

					parameters[tagKeyParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: "",
					}

					parameters[tagValueParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: "",
					}
				}

				assignmentWithParams.AssignmentProperties.Parameters = parameters

				assignments.EXPECT().Create(ctx, resourceGroup, displayName, assignmentWithParams).Return(assignmentWithSpID, nil)
			},
			roleAssignmentsMocks: func(roleAssignments *mock_authorization.MockRoleAssignmentsClient) {
				roleAssignments.EXPECT().Create(ctx, resourceGroup, uuid, roleAssignmentParams).Return(mgmtauthorization.RoleAssignment{}, nil)
			},
		},
		{
			name: "non-empty, non-maxxed ResourceTags in cluster doc - some policy parameters non-empty",
			modify: func(m *manager) {
				m.doc.OpenShiftCluster.Properties.ResourceTags = map[string]string{}

				for k, v := range tagSet {
					m.doc.OpenShiftCluster.Properties.ResourceTags[k] = v
				}
			},
			definitionsMocks: func(definitions *mock_policy.MockDefinitionsClient) {
				definitionWithID := definition
				definitionWithID.ID = to.StringPtr(definitionID)
				definitions.EXPECT().CreateOrUpdate(ctx, displayName, definition).Return(definitionWithID, nil)
			},
			assignmentsMocks: func(assignments *mock_policy.MockAssignmentsClient) {
				assignmentWithParams := assignment
				parameters := map[string]*mgmtpolicy.ParameterValuesValue{}

				for i := 0; i < len(tagSet); i++ {
					tagKeyParamName := tagKeyParamName(i)
					tagValueParamName := tagValueParamName(i)

					parameters[tagKeyParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: tagSetKeys[i],
					}

					parameters[tagValueParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: tagSet[tagSetKeys[i]],
					}
				}

				for i := len(tagSet); i < maxTags; i++ {
					tagKeyParamName := tagKeyParamName(i)
					tagValueParamName := tagValueParamName(i)

					parameters[tagKeyParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: "",
					}

					parameters[tagValueParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: "",
					}
				}

				assignmentWithParams.AssignmentProperties.Parameters = parameters

				assignments.EXPECT().Create(ctx, resourceGroup, displayName, assignmentWithParams).Return(assignmentWithSpID, nil)
			},
			roleAssignmentsMocks: func(roleAssignments *mock_authorization.MockRoleAssignmentsClient) {
				roleAssignments.EXPECT().Create(ctx, resourceGroup, uuid, roleAssignmentParams).Return(mgmtauthorization.RoleAssignment{}, nil)
			},
		},
		{
			name: "max length ResourceTags in cluster doc - no policy parameters empty",
			modify: func(m *manager) {
				m.doc.OpenShiftCluster.Properties.ResourceTags = map[string]string{}

				for k, v := range maxLenTagSet {
					m.doc.OpenShiftCluster.Properties.ResourceTags[k] = v
				}
			},
			definitionsMocks: func(definitions *mock_policy.MockDefinitionsClient) {
				definitionWithID := definition
				definitionWithID.ID = to.StringPtr(definitionID)
				definitions.EXPECT().CreateOrUpdate(ctx, displayName, definition).Return(definitionWithID, nil)
			},
			assignmentsMocks: func(assignments *mock_policy.MockAssignmentsClient) {
				assignmentWithParams := assignment
				parameters := map[string]*mgmtpolicy.ParameterValuesValue{}

				for i := 0; i < maxTags; i++ {
					tagKeyParamName := tagKeyParamName(i)
					tagValueParamName := tagValueParamName(i)

					parameters[tagKeyParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: maxLenTagSetKeys[i],
					}

					parameters[tagValueParamName] = &mgmtpolicy.ParameterValuesValue{
						Value: maxLenTagSet[maxLenTagSetKeys[i]],
					}
				}

				assignmentWithParams.AssignmentProperties.Parameters = parameters

				assignments.EXPECT().Create(ctx, resourceGroup, displayName, assignmentWithParams).Return(assignmentWithSpID, nil)
			},
			roleAssignmentsMocks: func(roleAssignments *mock_authorization.MockRoleAssignmentsClient) {
				roleAssignments.EXPECT().Create(ctx, resourceGroup, uuid, roleAssignmentParams).Return(mgmtauthorization.RoleAssignment{}, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			assignmentsClient := mock_policy.NewMockAssignmentsClient(controller)
			tt.assignmentsMocks(assignmentsClient)

			definitionsClient := mock_policy.NewMockDefinitionsClient(controller)
			tt.definitionsMocks(definitionsClient)

			roleAssignmentsClient := mock_authorization.NewMockRoleAssignmentsClient(controller)
			tt.roleAssignmentsMocks(roleAssignmentsClient)

			m := &manager{
				log:             logrus.NewEntry(logrus.StandardLogger()),
				assignments:     assignmentsClient,
				definitions:     definitionsClient,
				roleAssignments: roleAssignmentsClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroup,
							},
							InfraID: infraID,
							UUID:    uuid,
						},
					},
				},
			}

			if tt.modify != nil {
				tt.modify(m)
			}

			err := m.ensureTaggingPolicy(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestRemediateTags(t *testing.T) {
	ctx := context.Background()
	resourceGroupName := "fakeResourceGroup"
	resourceGroup := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/%s", resourceGroupName)

	baseTags := map[string]string{
		"foo":  "foo",
		"bar":  "bar",
		"bash": "bash",
		"zsh":  "zsh",
	}

	newTags := map[string]string{
		"new_tag":         "hi",
		"another_new_tag": "hello",
	}

	newTagsWithChanged := map[string]string{
		"zsh": "changed!",
	}

	for k, v := range newTags {
		newTagsWithChanged[k] = v
	}

	baseTagsWithPointers := map[string]*string{}

	for k, v := range baseTags {
		baseTagsWithPointers[k] = to.StringPtr(v)
	}

	azResourcesExpanded := []mgmtfeatures.GenericResourceExpanded{
		{
			Name: to.StringPtr("disk"),
			ID:   to.StringPtr("disk-id"),
			Tags: baseTagsWithPointers,
		},
		{
			Name: to.StringPtr("lb"),
			ID:   to.StringPtr("lb-id"),
			Tags: baseTagsWithPointers,
		},
		{
			Name: to.StringPtr("master-vm"),
			ID:   to.StringPtr("master-vm-id"),
			Tags: baseTagsWithPointers,
		},
		{
			Name: to.StringPtr("worker-vm"),
			ID:   to.StringPtr("worker-vm-id"),
			Tags: baseTagsWithPointers,
		},
		{
			Name: to.StringPtr("nsg"),
			ID:   to.StringPtr("nsg-id"),
			Tags: baseTagsWithPointers,
		},
	}

	azResourcesExpandedNoTags := []mgmtfeatures.GenericResourceExpanded{
		{
			Name: to.StringPtr("disk"),
			ID:   to.StringPtr("disk-id"),
		},
		{
			Name: to.StringPtr("lb"),
			ID:   to.StringPtr("lb-id"),
		},
		{
			Name: to.StringPtr("master-vm"),
			ID:   to.StringPtr("master-vm-id"),
		},
		{
			Name: to.StringPtr("worker-vm"),
			ID:   to.StringPtr("worker-vm-id"),
		},
		{
			Name: to.StringPtr("nsg"),
			ID:   to.StringPtr("nsg-id"),
		},
	}

	for _, tt := range []struct {
		name    string
		modify  func(*manager)
		mocks   func(*mock_features.MockResourcesClient)
		wantErr string
	}{
		{
			name: "empty ResourceTags - no tag updates needed",
		},
		{
			name: "non-empty ResourceTags - only adding new tags",
			modify: func(m *manager) {
				m.doc.OpenShiftCluster.Properties.ResourceTags = map[string]string{}

				for k, v := range newTags {
					m.doc.OpenShiftCluster.Properties.ResourceTags[k] = v
				}
			},
			mocks: func(resources *mock_features.MockResourcesClient) {
				resources.EXPECT().ListByResourceGroup(ctx, resourceGroupName, "", "", nil).Return(azResourcesExpanded, nil)

				mergedTags := map[string]*string{}

				for k, v := range baseTags {
					mergedTags[k] = to.StringPtr(v)
				}

				for k, v := range newTags {
					mergedTags[k] = to.StringPtr(v)
				}

				azResourceNewTags := mgmtfeatures.GenericResource{
					Tags: mergedTags,
				}

				resources.EXPECT().UpdateByIDAndWait(ctx, gomock.Any(), "2021-04-01", gomock.Eq(azResourceNewTags)).Return(nil).Times(len(azResourcesExpanded))
			},
		},
		{
			name: "non-empty ResourceTags - adding new tags and modifying one",
			modify: func(m *manager) {
				m.doc.OpenShiftCluster.Properties.ResourceTags = map[string]string{}

				for k, v := range newTagsWithChanged {
					m.doc.OpenShiftCluster.Properties.ResourceTags[k] = v
				}
			},
			mocks: func(resources *mock_features.MockResourcesClient) {
				resources.EXPECT().ListByResourceGroup(ctx, resourceGroupName, "", "", nil).Return(azResourcesExpanded, nil)

				mergedTags := map[string]*string{}

				for k, v := range baseTags {
					mergedTags[k] = to.StringPtr(v)
				}

				for k, v := range newTagsWithChanged {
					mergedTags[k] = to.StringPtr(v)
				}

				azResourceNewTags := mgmtfeatures.GenericResource{
					Tags: mergedTags,
				}

				resources.EXPECT().UpdateByIDAndWait(ctx, gomock.Any(), "2021-04-01", gomock.Eq(azResourceNewTags)).Return(nil).Times(len(azResourcesExpanded))
			},
		},
		{
			name: "non-empty ResourceTags - adding to nil Tags sets (on Azure resources)",
			modify: func(m *manager) {
				m.doc.OpenShiftCluster.Properties.ResourceTags = map[string]string{}

				for k, v := range newTags {
					m.doc.OpenShiftCluster.Properties.ResourceTags[k] = v
				}
			},
			mocks: func(resources *mock_features.MockResourcesClient) {
				resources.EXPECT().ListByResourceGroup(ctx, resourceGroupName, "", "", nil).Return(azResourcesExpandedNoTags, nil)

				_newTags := map[string]*string{}

				for k, v := range newTags {
					_newTags[k] = to.StringPtr(v)
				}

				azResourceNewTags := mgmtfeatures.GenericResource{
					Tags: _newTags,
				}

				resources.EXPECT().UpdateByIDAndWait(ctx, gomock.Any(), "2021-04-01", gomock.Eq(azResourceNewTags)).Return(nil).Times(5)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			resourcesClient := mock_features.NewMockResourcesClient(controller)

			if tt.mocks != nil {
				tt.mocks(resourcesClient)
			}

			m := &manager{
				log:       logrus.NewEntry(logrus.StandardLogger()),
				resources: resourcesClient,
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: resourceGroup,
							},
						},
					},
				},
			}

			if tt.modify != nil {
				tt.modify(m)
			}

			err := m.remediateTags(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestSetMasterSubnetPolicies(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		mocks          func(*mock_subnet.MockManager)
		gatewayEnabled bool
		wantErr        string
	}{
		{
			name: "ok, !gatewayEnabled",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "subnetID").Return(&mgmtnetwork.Subnet{}, nil)
				subnet.EXPECT().CreateOrUpdate(ctx, "subnetID", &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						PrivateLinkServiceNetworkPolicies: to.StringPtr("Disabled"),
					},
				}).Return(nil)
			},
		},
		{
			name: "ok, gatewayEnabled",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "subnetID").Return(&mgmtnetwork.Subnet{}, nil)
				subnet.EXPECT().CreateOrUpdate(ctx, "subnetID", &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						PrivateEndpointNetworkPolicies:    to.StringPtr("Disabled"),
						PrivateLinkServiceNetworkPolicies: to.StringPtr("Disabled"),
					},
				}).Return(nil)
			},
			gatewayEnabled: true,
		},
		{
			name: "error",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "subnetID").Return(nil, fmt.Errorf("sad"))
			},
			wantErr: "sad",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnet := mock_subnet.NewMockManager(controller)
			tt.mocks(subnet)

			m := &manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								SubnetID: "subnetID",
							},
							FeatureProfile: api.FeatureProfile{
								GatewayEnabled: tt.gatewayEnabled,
							},
						},
					},
				},
				subnet: subnet,
			}

			err := m.setMasterSubnetPolicies(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestEnsureInfraID(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name          string
		oc            *api.OpenShiftClusterDocument
		wantedInfraID string
		wantErr       string
	}{
		{
			name: "infra ID not set",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),

				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   resourceID,
					Name: "FoobarCluster",

					Properties: api.OpenShiftClusterProperties{
						InfraID: "",
					},
				},
			},
			wantedInfraID: "foobarcluster-cbhtc",
		},
		{
			name: "infra ID not set, very long name",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),

				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   resourceID,
					Name: "abcdefghijklmnopqrstuvwxyzabc",

					Properties: api.OpenShiftClusterProperties{
						InfraID: "",
					},
				},
			},
			wantedInfraID: "abcdefghijklmnopqrstu-cbhtc",
		},
		{
			name: "infra ID set and left alone",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),

				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   resourceID,
					Name: "FoobarCluster",

					Properties: api.OpenShiftClusterProperties{
						InfraID: "infra",
					},
				},
			},
			wantedInfraID: "infra",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

			f := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			f.AddOpenShiftClusterDocuments(tt.oc)

			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				db:  dbOpenShiftClusters,
				doc: doc,
			}

			// hopefully setting a seed here means it passes consistently :)
			utilrand.Seed(0)
			err = m.ensureInfraID(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			checkDoc, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}

			if checkDoc.OpenShiftCluster.Properties.InfraID != tt.wantedInfraID {
				t.Fatalf("%s != %s (wanted)", checkDoc.OpenShiftCluster.Properties.InfraID, tt.wantedInfraID)
			}
		})
	}
}

func TestEnsureUUID(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftClusterDocument
		wantErr string
	}{
		{
			name: "UUID not set",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),

				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   resourceID,
					Name: "FoobarCluster",

					Properties: api.OpenShiftClusterProperties{
						UUID: "",
					},
				},
			},
		},
		{
			name: "UUID set",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),

				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   resourceID,
					Name: "FoobarCluster",

					Properties: api.OpenShiftClusterProperties{
						UUID: "123e4567-e89b-12d3-a456-426614174000",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

			f := testdatabase.NewFixture().WithOpenShiftClusters(dbOpenShiftClusters)
			f.AddOpenShiftClusterDocuments(tt.oc)

			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			doc, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}

			m := &manager{
				db:  dbOpenShiftClusters,
				doc: doc,
			}

			err = m.ensureUUID(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			checkDoc, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}

			if !uuid.IsValid(checkDoc.OpenShiftCluster.Properties.UUID) {
				t.Fatalf("Invalid UUID %s", checkDoc.OpenShiftCluster.Properties.UUID)
			}
		})
	}
}

// modifyManagerResourceTags is a helper function used by some test cases to tweak the set of
// tags included in an OpenShiftClusterDocument.
func modifyManagerResourceTags(m *manager) {
	m.doc.OpenShiftCluster.Properties.ResourceTags = map[string]string{
		"foo": "bar",
		"bar": "baz",
	}
}
