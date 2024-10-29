package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestDenyAssignment(t *testing.T) {
	m := &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	tests := []struct {
		Name                      string
		ClusterDocument           *api.OpenShiftClusterDocument
		ExpectedExcludePrincipals *[]mgmtauthorization.Principal
	}{
		{
			Name: "cluster with ServicePrincipalProfile",
			ClusterDocument: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster",
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							SPObjectID: fakeClusterSPObjectId,
						},
					},
				},
			},
			ExpectedExcludePrincipals: &[]mgmtauthorization.Principal{
				{
					ID:   to.StringPtr(fakeClusterSPObjectId),
					Type: to.StringPtr(string(mgmtauthorization.ServicePrincipal)),
				},
			},
		},
		{
			Name: "cluster with PlatformWorkloadIdentityProfile",
			ClusterDocument: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster",
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
			ExpectedExcludePrincipals: &[]mgmtauthorization.Principal{
				{
					ID:   to.StringPtr("00000000-0000-0000-0000-000000000000"),
					Type: to.StringPtr(string(mgmtauthorization.ServicePrincipal)),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			m.doc = test.ClusterDocument

			actualDenyAssignment := m.denyAssignment().Resource.(*mgmtauthorization.DenyAssignment)
			actualExcludePrincipals := actualDenyAssignment.ExcludePrincipals

			if !reflect.DeepEqual(test.ExpectedExcludePrincipals, actualExcludePrincipals) {
				t.Errorf("expected %+v, got %+v\n", test.ExpectedExcludePrincipals, actualExcludePrincipals)
			}
		})
	}
}

func TestFpspStorageBlobContributorRBAC(t *testing.T) {
	storageAccountName := "clustertest"
	fakePrincipalID := "fakeID"
	resourceType := "Microsoft.Storage/storageAccounts"
	resourceID := fmt.Sprintf("resourceId('%s', '%s')", resourceType, storageAccountName)
	tests := []struct {
		Name                string
		ClusterDocument     *api.OpenShiftClusterDocument
		ExpectedArmResource *arm.Resource
		wantErr             string
	}{
		{
			Name: "Fail : cluster with ServicePrincipalProfile",
			ClusterDocument: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster",
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							SPObjectID: fakeClusterSPObjectId,
						},
					},
				},
			},
			wantErr: "fpspStorageBlobContributorRBAC called for a Cluster Service Principal cluster",
		},
		{
			Name: "Success : cluster with PlatformWorkloadIdentityProfile",
			ClusterDocument: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
				},
			},
			ExpectedArmResource: &arm.Resource{
				Resource: mgmtauthorization.RoleAssignment{
					Name: to.StringPtr("[concat('clustertest', '/Microsoft.Authorization/', guid(" + resourceID + "))]"),
					Type: to.StringPtr(resourceType + "/providers/roleAssignments"),
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            to.StringPtr("[" + resourceID + "]"),
						RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '" + rbac.RoleStorageBlobDataContributor + "')]"),
						PrincipalID:      to.StringPtr("['" + fakePrincipalID + "']"),
						PrincipalType:    mgmtauthorization.ServicePrincipal,
					},
				},
				APIVersion: azureclient.APIVersion("Microsoft.Authorization"),
				DependsOn: []string{
					"[" + resourceID + "]",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)

			m := &manager{
				doc: tt.ClusterDocument,
				env: env,
			}
			resource, err := m.fpspStorageBlobContributorRBAC(storageAccountName, fakePrincipalID)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if !reflect.DeepEqual(tt.ExpectedArmResource, resource) {
				t.Error("resultant ARM resource isn't the same as expected.")
			}
		})
	}
}
