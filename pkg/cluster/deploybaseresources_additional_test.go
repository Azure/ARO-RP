package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	uuidfake "github.com/Azure/ARO-RP/pkg/util/uuid/fake"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestDenyAssignment(t *testing.T) {
	m := &manager{
		log:                  logrus.NewEntry(logrus.StandardLogger()),
		fpServicePrincipalID: "77777777-7777-7777-7777-777777777777",
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
					ID:   pointerutils.ToPtr(fakeClusterSPObjectId),
					Type: pointerutils.ToPtr(string(mgmtauthorization.ServicePrincipal)),
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
								"something other than anything": {
									ObjectID:   "88888888-8888-8888-8888-888888888888",
									ClientID:   "99999999-9999-9999-9999-999999999999",
									ResourceID: "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/something/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name",
								},
							},
						},
					},
				},
			},
			ExpectedExcludePrincipals: &[]mgmtauthorization.Principal{
				{
					ID:   pointerutils.ToPtr("00000000-0000-0000-0000-000000000000"),
					Type: pointerutils.ToPtr(string(mgmtauthorization.ServicePrincipal)),
				},
				{
					ID:   pointerutils.ToPtr("88888888-8888-8888-8888-888888888888"),
					Type: pointerutils.ToPtr(string(mgmtauthorization.ServicePrincipal)),
				},
				{
					ID:   pointerutils.ToPtr("77777777-7777-7777-7777-777777777777"),
					Type: pointerutils.ToPtr(string(mgmtauthorization.ServicePrincipal)),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			m.doc = test.ClusterDocument

			actualDenyAssignment := m.denyAssignment().Resource.(*mgmtauthorization.DenyAssignment)
			actualExcludePrincipals := actualDenyAssignment.ExcludePrincipals

			// Sort the principals coming back before we compare them
			sortfunc := func(a mgmtauthorization.Principal, b mgmtauthorization.Principal) int {
				return strings.Compare(*a.ID, *b.ID)
			}
			slices.SortFunc(*actualExcludePrincipals, sortfunc)
			slices.SortFunc(*test.ExpectedExcludePrincipals, sortfunc)

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
					Name: pointerutils.ToPtr("[concat('clustertest', '/Microsoft.Authorization/', guid(" + resourceID + "))]"),
					Type: pointerutils.ToPtr(resourceType + "/providers/roleAssignments"),
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            pointerutils.ToPtr("[" + resourceID + "]"),
						RoleDefinitionID: pointerutils.ToPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '" + rbac.RoleStorageBlobDataContributor + "')]"),
						PrincipalID:      pointerutils.ToPtr("['" + fakePrincipalID + "']"),
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

func TestNetworkInternalLoadBalancerZonality(t *testing.T) {
	infraID := "infraID"
	location := "eastus"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	// Define the DB instance we will use to run the PatchWithLease function
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	// Run tests
	for _, tt := range []struct {
		name                string
		m                   manager
		expectedARMResource *arm.Resource
		uuids               []string
	}{
		{
			name:  "non-zonal",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV2,
							ProvisioningState:   api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							MasterProfile: api.MasterProfile{
								SubnetID: "someID",
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								LoadBalancerProfile: &api.LoadBalancerProfile{},
							},
						},
					},
				},
			},
			expectedARMResource: &arm.Resource{
				Resource: &armnetwork.LoadBalancer{
					SKU: &armnetwork.LoadBalancerSKU{
						Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameStandard),
					},
					Properties: &armnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
							{
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
									Subnet: &armnetwork.Subnet{
										ID: pointerutils.ToPtr("someID"),
									},
								},
								Zones: []*string{},
								Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
							},
						},
						BackendAddressPools: []*armnetwork.BackendAddressPool{
							{
								Name: pointerutils.ToPtr(infraID),
							},
							{
								Name: pointerutils.ToPtr("ssh-0"),
							},
							{
								Name: pointerutils.ToPtr("ssh-1"),
							},
							{
								Name: pointerutils.ToPtr("ssh-2"),
							},
						},
					},
					Name:     pointerutils.ToPtr(infraID + "-internal"),
					Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
					Location: pointerutils.ToPtr(location),
				},
				APIVersion: azureclient.APIVersion("Microsoft.Network"),
				DependsOn:  []string{},
			},
		},
		{
			name:  "zonal",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ArchitectureVersion: api.ArchitectureVersionV2,
							ProvisioningState:   api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							MasterProfile: api.MasterProfile{
								SubnetID: "someID",
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							NetworkProfile: api.NetworkProfile{
								LoadBalancerProfile: &api.LoadBalancerProfile{
									Zones: []string{"1", "2", "3"},
								},
							},
						},
					},
				},
			},
			expectedARMResource: &arm.Resource{
				Resource: &armnetwork.LoadBalancer{
					SKU: &armnetwork.LoadBalancerSKU{
						Name: pointerutils.ToPtr(armnetwork.LoadBalancerSKUNameStandard),
					},
					Properties: &armnetwork.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
							{
								Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
									PrivateIPAllocationMethod: pointerutils.ToPtr(armnetwork.IPAllocationMethodDynamic),
									Subnet: &armnetwork.Subnet{
										ID: pointerutils.ToPtr("someID"),
									},
								},
								Zones: []*string{pointerutils.ToPtr("1"), pointerutils.ToPtr("2"), pointerutils.ToPtr("3")},
								Name:  pointerutils.ToPtr("internal-lb-ip-v4"),
							},
						},
						BackendAddressPools: []*armnetwork.BackendAddressPool{
							{
								Name: pointerutils.ToPtr(infraID),
							},
							{
								Name: pointerutils.ToPtr("ssh-0"),
							},
							{
								Name: pointerutils.ToPtr("ssh-1"),
							},
							{
								Name: pointerutils.ToPtr("ssh-2"),
							},
						},
					},
					Name:     pointerutils.ToPtr(infraID + "-internal"),
					Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
					Location: pointerutils.ToPtr(location),
				},
				APIVersion: azureclient.APIVersion("Microsoft.Network"),
				DependsOn:  []string{},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Create the DB to test the cluster
			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.m.doc)
			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}
			tt.m.db = openShiftClustersDatabase
			tt.m.log = logrus.NewEntry(logrus.StandardLogger())

			uuid.DefaultGenerator = uuidfake.NewGenerator(tt.uuids)

			resource := tt.m.networkInternalLoadBalancer(location)

			// nil out the LB rules since we don't test them here
			lb := resource.Resource.(*armnetwork.LoadBalancer)
			lb.Properties.Probes = nil
			lb.Properties.LoadBalancingRules = nil

			if !assert.Equal(t, tt.expectedARMResource, resource) {
				for _, x := range deep.Equal(tt.expectedARMResource, resource) {
					t.Log(x)
				}
			}
		})
	}
}
