package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	azstorage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_azblob "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azblob"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_blob "github.com/Azure/ARO-RP/pkg/util/mocks/blob"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/oidcbuilder"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	uuidfake "github.com/Azure/ARO-RP/pkg/util/uuid/fake"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilmsi "github.com/Azure/ARO-RP/test/util/azure/msi"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
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

	disallowedByPolicy := autorest.NewErrorWithError(&azure.RequestError{
		ServiceError: &azure.ServiceError{Code: "RequestDisallowedByPolicy"},
	}, "", "", nil, "")

	resourceGroupNotFound := autorest.NewErrorWithError(&azure.RequestError{
		ServiceError: &azure.ServiceError{Code: "ResourceGroupNotFound"},
	}, "", "", &http.Response{StatusCode: http.StatusNotFound}, "")

	for _, tt := range []struct {
		name              string
		provisioningState api.ProvisioningState
		mocks             func(*mock_features.MockResourceGroupsClient, *mock_env.MockInterface)
		wantErr           string
	}{
		{
			name:              "success - rg doesn't exist",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					Get(gomock.Any(), resourceGroupName).
					Return(mgmtfeatures.ResourceGroup{}, resourceGroupNotFound)

				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, group).
					Return(group, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(false)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), resourceGroupName).
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
					Get(gomock.Any(), resourceGroupName).
					Return(mgmtfeatures.ResourceGroup{}, resourceGroupNotFound)

				rg.EXPECT().
					CreateOrUpdate(gomock.Any(), resourceGroupName, groupWithLocalDevTags).
					Return(groupWithLocalDevTags, nil)

				env.EXPECT().
					IsLocalDevelopmentMode().
					Return(true)

				env.EXPECT().
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), resourceGroupName).
					Return(nil)
			},
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
					EnsureARMResourceGroupRoleAssignment(gomock.Any(), resourceGroupName).
					Return(nil)
			},
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
			name:              "fail - managedBy doesn't match",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				badManagedBy := group
				badManagedBy.ManagedBy = to.StringPtr("does-not-match")
				rg.EXPECT().
					Get(gomock.Any(), resourceGroupName).
					Return(badManagedBy, nil)
			},
			wantErr: "400: ClusterResourceGroupAlreadyExists: : Resource group " + resourceGroup + " must not already exist.",
		},
		{
			name:              "fail - location doesn't match",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				badLocation := group
				badLocation.Location = to.StringPtr("bad-location")
				rg.EXPECT().
					Get(gomock.Any(), resourceGroupName).
					Return(badLocation, nil)
			},
			wantErr: "400: ClusterResourceGroupAlreadyExists: : Resource group " + resourceGroup + " must not already exist.",
		},
		{
			name:              "fail - CreateOrUpdate returns requestdisallowedbypolicy",
			provisioningState: api.ProvisioningStateCreating,
			mocks: func(rg *mock_features.MockResourceGroupsClient, env *mock_env.MockInterface) {
				rg.EXPECT().
					Get(gomock.Any(), resourceGroupName).
					Return(group, nil)

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
					Get(gomock.Any(), resourceGroupName).
					Return(group, nil)

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

			err := m.ensureResourceGroup(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestAttachNSGs(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftClusterDocument
		mocks   func(*mock_subnet.MockManager)
		wantErr string
	}{
		{
			name: "Success - NSG attached to both subnets",
			oc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: api.ArchitectureVersionV2,
						InfraID:             "infra",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678",
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "masterSubnetID",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								SubnetID: "workerSubnetID",
							},
						},
					},
				},
			},
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "masterSubnetID").Return(&mgmtnetwork.Subnet{}, nil)
				subnet.EXPECT().CreateOrUpdate(ctx, "masterSubnetID", &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678/providers/Microsoft.Network/networkSecurityGroups/infra-nsg"),
						},
					},
				}).Return(nil)
				subnet.EXPECT().Get(ctx, "workerSubnetID").Return(&mgmtnetwork.Subnet{}, nil)
				subnet.EXPECT().CreateOrUpdate(ctx, "workerSubnetID", &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678/providers/Microsoft.Network/networkSecurityGroups/infra-nsg"),
						},
					},
				}).Return(nil)
			},
		},
		{
			name: "Success - preconfigured NSG enabled",
			oc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: api.ArchitectureVersionV2,
						InfraID:             "infra",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678",
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "masterSubnetID",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								SubnetID: "workerSubnetID",
							},
						},
						NetworkProfile: api.NetworkProfile{
							PreconfiguredNSG: api.PreconfiguredNSGEnabled,
						},
					},
				},
			},
			mocks: func(subnet *mock_subnet.MockManager) {},
		},
		{
			name: "Failure - unable to get a subnet",
			oc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: api.ArchitectureVersionV2,
						InfraID:             "infra",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678",
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "masterSubnetID",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								SubnetID: "workerSubnetID",
							},
						},
					},
				},
			},
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "masterSubnetID").Return(&mgmtnetwork.Subnet{}, fmt.Errorf("subnet not found"))
			},
			wantErr: "subnet not found",
		},
		{
			name: "Failure - NSG already attached to a subnet",
			oc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: api.ArchitectureVersionV2,
						InfraID:             "infra",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678",
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "masterSubnetID",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								SubnetID: "workerSubnetID",
							},
						},
					},
				},
			},
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "masterSubnetID").Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: to.StringPtr("I shouldn't be here!"),
						},
					},
				}, nil)
			},
			wantErr: "400: InvalidLinkedVNet: : The provided subnet 'masterSubnetID' is invalid: must not have a network security group attached.",
		},
		{
			name: "Failure - failed to CreateOrUpdate subnet because NSG not yet ready for use",
			oc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: api.ArchitectureVersionV2,
						InfraID:             "infra",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678",
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "masterSubnetID",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								SubnetID: "workerSubnetID",
							},
						},
					},
				},
			},
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "masterSubnetID").Return(&mgmtnetwork.Subnet{}, nil)
				subnet.EXPECT().CreateOrUpdate(ctx, "masterSubnetID", &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678/providers/Microsoft.Network/networkSecurityGroups/infra-nsg"),
						},
					},
				}).Return(fmt.Errorf("Some random stuff followed by the important part that we're trying to match: Resource /subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678/providers/Microsoft.Network/networkSecurityGroups/infra-nsg referenced by resource masterSubnetID was not found. and here's some more stuff that's at the end past the important part"))
			},
		},
		{
			name: "Failure - failed to CreateOrUpdate subnet with arbitrary error",
			oc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ArchitectureVersion: api.ArchitectureVersionV2,
						InfraID:             "infra",
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678",
						},
						MasterProfile: api.MasterProfile{
							SubnetID: "masterSubnetID",
						},
						WorkerProfiles: []api.WorkerProfile{
							{
								SubnetID: "workerSubnetID",
							},
						},
					},
				},
			},
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, "masterSubnetID").Return(&mgmtnetwork.Subnet{}, nil)
				subnet.EXPECT().CreateOrUpdate(ctx, "masterSubnetID", &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/aro-12345678/providers/Microsoft.Network/networkSecurityGroups/infra-nsg"),
						},
					},
				}).Return(fmt.Errorf("I'm an arbitrary error here to make life harder"))
			},
			wantErr: "I'm an arbitrary error here to make life harder",
		},
	} {
		controller := gomock.NewController(t)
		defer controller.Finish()

		subnet := mock_subnet.NewMockManager(controller)
		tt.mocks(subnet)

		m := &manager{
			log:    logrus.NewEntry(logrus.StandardLogger()),
			doc:    tt.oc,
			subnet: subnet,
		}

		err := m._attachNSGs(ctx, 1*time.Millisecond, 30*time.Second)
		utilerror.AssertErrorMessage(t, err, tt.wantErr)
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

func TestSubnetsWithServiceEndpoints(t *testing.T) {
	ctx := context.Background()
	masterSubnet := strings.ToLower("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master-subnet")
	workerSubnetFormatString := strings.ToLower("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/%s")
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"
	serviceEndpoint := "Microsoft.Storage"
	location := "eastus"

	for _, tt := range []struct {
		name          string
		mocks         func(subnet *mock_subnet.MockManager)
		workerSubnets []string
		wantSubnets   []string
		wantErr       string
	}{
		{
			name: "no service endpoints set returns empty string slice",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, masterSubnet).Return(&mgmtnetwork.Subnet{}, nil)
			},
			wantSubnets: []string{},
		},
		{
			name: "master subnet has service endpoint, but incorrect location",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, masterSubnet).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service: &serviceEndpoint,
								Locations: &[]string{
									"bad-location",
								},
							},
						},
					},
				}, nil)
				subnet.EXPECT().Get(ctx, fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001")).Return(&mgmtnetwork.Subnet{}, nil)
			},
			workerSubnets: []string{
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001"),
			},
			wantSubnets: []string{},
		},
		{
			name: "master subnet has service endpoint with correct location",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, masterSubnet).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service: &serviceEndpoint,
								Locations: &[]string{
									location,
								},
							},
						},
					},
				}, nil)
				subnet.EXPECT().Get(ctx, fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001")).Return(&mgmtnetwork.Subnet{}, nil)
			},
			workerSubnets: []string{
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001"),
			},
			wantSubnets: []string{masterSubnet},
		},
		{
			name: "master subnet has service endpoint with all location",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, masterSubnet).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service: &serviceEndpoint,
								Locations: &[]string{
									"*",
								},
							},
						},
					},
				}, nil)
				subnet.EXPECT().Get(ctx, fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001")).Return(&mgmtnetwork.Subnet{}, nil)
			},
			workerSubnets: []string{
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001"),
			},
			wantSubnets: []string{masterSubnet},
		},
		{
			name: "all subnets have service endpoint with correct locations",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnetWithServiceEndpoint := &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service: &serviceEndpoint,
								Locations: &[]string{
									"*",
								},
							},
						},
					},
				}

				subnet.EXPECT().Get(ctx, masterSubnet).Return(subnetWithServiceEndpoint, nil)
				subnet.EXPECT().Get(ctx, fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001")).Return(subnetWithServiceEndpoint, nil)
			},
			workerSubnets: []string{
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001"),
			},
			wantSubnets: []string{
				masterSubnet,
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001"),
			},
		},
		{
			name: "mixed subnets with service endpoint",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnetWithServiceEndpoint := &mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service: &serviceEndpoint,
								Locations: &[]string{
									location,
								},
							},
						},
					},
				}

				subnet.EXPECT().Get(ctx, masterSubnet).Return(subnetWithServiceEndpoint, nil)
				subnet.EXPECT().Get(ctx, fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001")).Return(subnetWithServiceEndpoint, nil)
				subnet.EXPECT().Get(ctx, fmt.Sprintf(workerSubnetFormatString, "worker-subnet-002")).Return(&mgmtnetwork.Subnet{}, nil)
			},
			workerSubnets: []string{
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001"),
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-002"),
				"",
			},
			wantSubnets: []string{
				masterSubnet,
				fmt.Sprintf(workerSubnetFormatString, "worker-subnet-001"),
			},
		},
		{
			name: "Get subnet returns error",
			mocks: func(subnet *mock_subnet.MockManager) {
				subnet.EXPECT().Get(ctx, masterSubnet).Return(nil, errors.New("generic error"))
			},
			workerSubnets: []string{},
			wantErr:       "generic error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnet := mock_subnet.NewMockManager(controller)
			tt.mocks(subnet)

			workerProfiles := []api.WorkerProfile{}
			if tt.workerSubnets != nil {
				for _, subnet := range tt.workerSubnets {
					workerProfiles = append(workerProfiles, api.WorkerProfile{
						SubnetID: subnet,
					})
				}
			}

			m := &manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(resourceID),

					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       resourceID,
						Name:     "FoobarCluster",
						Location: location,

						Properties: api.OpenShiftClusterProperties{
							MasterProfile: api.MasterProfile{
								SubnetID: masterSubnet,
							},
							WorkerProfiles: workerProfiles,
						},
					},
				},
				subnet: subnet,
			}

			subnets, err := m.subnetsWithServiceEndpoint(ctx, serviceEndpoint)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			// sort slices for ordering
			sort.Strings(subnets)
			sort.Strings(tt.wantSubnets)

			if !reflect.DeepEqual(subnets, tt.wantSubnets) {
				t.Errorf("got: %v, wanted %v", subnets, tt.wantSubnets)
			}
		})
	}
}

func TestNewPublicLoadBalancer(t *testing.T) {
	ctx := context.Background()
	infraID := "infraID"
	location := "eastus"
	clusterRGID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG"
	// Define the DB instance we will use to run the PatchWithLease function
	key := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"

	// Run tests
	for _, tt := range []struct {
		name                 string
		m                    manager
		expectedARMResources []*arm.Resource
		uuids                []string
	}{
		{
			name:  "api server visibility public with 1 managed IP",
			uuids: []string{},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPrivate,
								},
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
						},
					},
				},
			},
			expectedARMResources: []*arm.Resource{
				{
					Resource: &mgmtnetwork.PublicIPAddress{
						Sku: &mgmtnetwork.PublicIPAddressSku{
							Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: mgmtnetwork.Static,
						},
						Name:     to.StringPtr(infraID + "-pip-v4"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &location,
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
				},
				{
					Resource: &mgmtnetwork.LoadBalancer{
						Sku: &mgmtnetwork.LoadBalancerSku{
							Name: mgmtnetwork.LoadBalancerSkuNameStandard,
						},
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
								{
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', '" + infraID + "-pip-v4')]"),
										},
									},
									Name: to.StringPtr("public-lb-ip-v4"),
								},
							},
							BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									Name: to.StringPtr(infraID),
								},
							},
							LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
								{
									LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s', 'public-lb-ip-v4')]", infraID)),
										},
										BackendAddressPool: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
										},
										Probe: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s', 'api-internal-probe')]", infraID)),
										},
										Protocol:             mgmtnetwork.TransportProtocolTCP,
										LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
										FrontendPort:         to.Int32Ptr(6443),
										BackendPort:          to.Int32Ptr(6443),
										IdleTimeoutInMinutes: to.Int32Ptr(30),
										DisableOutboundSnat:  to.BoolPtr(true),
									},
									Name: to.StringPtr("api-internal-v4"),
								},
							},
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(5),
										NumberOfProbes:    to.Int32Ptr(2),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
							},
							OutboundRules: &[]mgmtnetwork.OutboundRule{
								{
									OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
										FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
											{
												ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
											},
										},
										BackendAddressPool: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
										},
										Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
										IdleTimeoutInMinutes: to.Int32Ptr(30),
									},
									Name: to.StringPtr("outbound-rule-v4"),
								},
							},
						},
						Name:     to.StringPtr(infraID),
						Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
						Location: to.StringPtr(location),
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
					DependsOn: []string{
						"Microsoft.Network/publicIPAddresses/" + infraID + "-pip-v4",
					},
				},
			},
		},
		{
			name:  "api server visibility public with 2 managed IPs",
			uuids: []string{"uuid1"},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPublic,
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPrivate,
								},
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 2,
									},
								},
							},
						},
					},
				},
			},
			expectedARMResources: []*arm.Resource{
				{
					Resource: &mgmtnetwork.PublicIPAddress{
						Sku: &mgmtnetwork.PublicIPAddressSku{
							Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: mgmtnetwork.Static,
						},
						Name:     to.StringPtr(infraID + "-pip-v4"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &location,
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
				},
				{
					Resource: &mgmtnetwork.PublicIPAddress{
						Sku: &mgmtnetwork.PublicIPAddressSku{
							Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: mgmtnetwork.Static,
						},
						Name:     to.StringPtr("uuid1-outbound-pip-v4"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &location,
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
				},
				{
					Resource: &mgmtnetwork.LoadBalancer{
						Sku: &mgmtnetwork.LoadBalancerSku{
							Name: mgmtnetwork.LoadBalancerSkuNameStandard,
						},
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
								{
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', '" + infraID + "-pip-v4')]"),
										},
									},
									Name: to.StringPtr("public-lb-ip-v4"),
								},
								{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4"),
										},
									},
									Name: to.StringPtr("uuid1-outbound-pip-v4"),
								},
							},
							BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									Name: to.StringPtr(infraID),
								},
							},
							LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
								{
									LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
										FrontendIPConfiguration: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s', 'public-lb-ip-v4')]", infraID)),
										},
										BackendAddressPool: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
										},
										Probe: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s', 'api-internal-probe')]", infraID)),
										},
										Protocol:             mgmtnetwork.TransportProtocolTCP,
										LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
										FrontendPort:         to.Int32Ptr(6443),
										BackendPort:          to.Int32Ptr(6443),
										IdleTimeoutInMinutes: to.Int32Ptr(30),
										DisableOutboundSnat:  to.BoolPtr(true),
									},
									Name: to.StringPtr("api-internal-v4"),
								},
							},
							Probes: &[]mgmtnetwork.Probe{
								{
									ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
										Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
										Port:              to.Int32Ptr(6443),
										IntervalInSeconds: to.Int32Ptr(5),
										NumberOfProbes:    to.Int32Ptr(2),
										RequestPath:       to.StringPtr("/readyz"),
									},
									Name: to.StringPtr("api-internal-probe"),
								},
							},
							OutboundRules: &[]mgmtnetwork.OutboundRule{
								{
									OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
										FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
											{
												ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/public-lb-ip-v4"),
											},
											{
												ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
											},
										},
										BackendAddressPool: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
										},
										Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
										IdleTimeoutInMinutes: to.Int32Ptr(30),
									},
									Name: to.StringPtr("outbound-rule-v4"),
								},
							},
						},
						Name:     to.StringPtr(infraID),
						Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
						Location: to.StringPtr(location),
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
					DependsOn: []string{
						"Microsoft.Network/publicIPAddresses/" + infraID + "-pip-v4",
						"Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
					},
				},
			},
		},
		{
			name:  "api server visibility private with 1 managed IP",
			uuids: []string{"uuid1"},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPrivate,
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPrivate,
								},
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 1,
									},
								},
							},
						},
					},
				},
			},
			expectedARMResources: []*arm.Resource{
				{
					Resource: &mgmtnetwork.PublicIPAddress{
						Sku: &mgmtnetwork.PublicIPAddressSku{
							Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: mgmtnetwork.Static,
						},
						Name:     to.StringPtr("uuid1-outbound-pip-v4"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &location,
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
				},
				{
					Resource: &mgmtnetwork.LoadBalancer{
						Sku: &mgmtnetwork.LoadBalancerSku{
							Name: mgmtnetwork.LoadBalancerSkuNameStandard,
						},
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
								{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4"),
										},
									},
									Name: to.StringPtr("uuid1-outbound-pip-v4"),
								},
							},
							BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									Name: to.StringPtr(infraID),
								},
							},
							LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{},
							Probes:             &[]mgmtnetwork.Probe{},
							OutboundRules: &[]mgmtnetwork.OutboundRule{
								{
									OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
										FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
											{
												ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
											},
										},
										BackendAddressPool: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
										},
										Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
										IdleTimeoutInMinutes: to.Int32Ptr(30),
									},
									Name: to.StringPtr("outbound-rule-v4"),
								},
							},
						},
						Name:     to.StringPtr(infraID),
						Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
						Location: to.StringPtr(location),
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
					DependsOn: []string{
						"Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
					},
				},
			},
		},
		{
			name:  "api server visibility private with 2 managed IPs",
			uuids: []string{"uuid1", "uuid2"},
			m: manager{
				doc: &api.OpenShiftClusterDocument{
					Key: strings.ToLower(key),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       key,
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterRGID,
							},
							InfraID: infraID,
							APIServerProfile: api.APIServerProfile{
								Visibility: api.VisibilityPrivate,
							},
							IngressProfiles: []api.IngressProfile{
								{
									Visibility: api.VisibilityPrivate,
								},
							},
							NetworkProfile: api.NetworkProfile{
								OutboundType: api.OutboundTypeLoadbalancer,
								LoadBalancerProfile: &api.LoadBalancerProfile{
									ManagedOutboundIPs: &api.ManagedOutboundIPs{
										Count: 2,
									},
								},
							},
						},
					},
				},
			},
			expectedARMResources: []*arm.Resource{
				{
					Resource: &mgmtnetwork.PublicIPAddress{
						Sku: &mgmtnetwork.PublicIPAddressSku{
							Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: mgmtnetwork.Static,
						},
						Name:     to.StringPtr("uuid1-outbound-pip-v4"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &location,
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
				},
				{
					Resource: &mgmtnetwork.PublicIPAddress{
						Sku: &mgmtnetwork.PublicIPAddressSku{
							Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
						},
						PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
							PublicIPAllocationMethod: mgmtnetwork.Static,
						},
						Name:     to.StringPtr("uuid2-outbound-pip-v4"),
						Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
						Location: &location,
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
				},
				{
					Resource: &mgmtnetwork.LoadBalancer{
						Sku: &mgmtnetwork.LoadBalancerSku{
							Name: mgmtnetwork.LoadBalancerSkuNameStandard,
						},
						LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
							FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
								{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4"),
										},
									},
									Name: to.StringPtr("uuid1-outbound-pip-v4"),
								},
								{
									ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid2-outbound-pip-v4"),
									FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
										PublicIPAddress: &mgmtnetwork.PublicIPAddress{
											ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/publicIPAddresses/uuid2-outbound-pip-v4"),
										},
									},
									Name: to.StringPtr("uuid2-outbound-pip-v4"),
								},
							},
							BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
								{
									Name: to.StringPtr(infraID),
								},
							},
							LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{},
							Probes:             &[]mgmtnetwork.Probe{},
							OutboundRules: &[]mgmtnetwork.OutboundRule{
								{
									OutboundRulePropertiesFormat: &mgmtnetwork.OutboundRulePropertiesFormat{
										FrontendIPConfigurations: &[]mgmtnetwork.SubResource{
											{
												ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid1-outbound-pip-v4"),
											},
											{
												ID: to.StringPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/clusterRG/providers/Microsoft.Network/loadBalancers/infraID/frontendIPConfigurations/uuid2-outbound-pip-v4"),
											},
										},
										BackendAddressPool: &mgmtnetwork.SubResource{
											ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", infraID)),
										},
										Protocol:             mgmtnetwork.LoadBalancerOutboundRuleProtocolAll,
										IdleTimeoutInMinutes: to.Int32Ptr(30),
									},
									Name: to.StringPtr("outbound-rule-v4"),
								},
							},
						},
						Name:     to.StringPtr(infraID),
						Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
						Location: to.StringPtr(location),
					},
					APIVersion: azureclient.APIVersion("Microsoft.Network"),
					DependsOn: []string{
						"Microsoft.Network/publicIPAddresses/uuid1-outbound-pip-v4",
						"Microsoft.Network/publicIPAddresses/uuid2-outbound-pip-v4",
					},
				},
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

			resources := []*arm.Resource{}
			tt.m.newPublicLoadBalancer(ctx, &resources)

			assert.Equal(t, tt.expectedARMResources, resources)
		})
	}
}

func TestCreateOIDC(t *testing.T) {
	ctx := context.Background()
	clusterID := "test-cluster"
	resourceGroupName := "fakeResourceGroup"
	oidcStorageAccountName := "eastusoic"
	afdEndpoint := "fake.oic.aro.test.net"
	tenantId := "00000000-0000-0000-0000-000000000000"
	m := manager{
		subscriptionDoc: &api.SubscriptionDocument{
			Subscription: &api.Subscription{
				Properties: &api.SubscriptionProperties{
					TenantID: tenantId,
				},
			},
		},
	}
	storageWebEndpointForDev := oidcStorageAccountName + ".web." + azureclient.PublicCloud.StorageEndpointSuffix
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName"
	blobContainerURL := fmt.Sprintf("https://%s.blob.%s/%s", oidcStorageAccountName, azureclient.PublicCloud.StorageEndpointSuffix, oidcbuilder.WebContainer)
	prodOIDCIssuer := fmt.Sprintf("https://%s/%s", afdEndpoint, oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, clusterID))
	devOIDCIssuer := fmt.Sprintf("https://%s/%s", storageWebEndpointForDev, oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, clusterID))
	containerProperties := azstorage.AccountsClientGetPropertiesResponse{
		Account: azstorage.Account{
			Properties: &azstorage.AccountProperties{
				PrimaryEndpoints: &azstorage.Endpoints{
					Web: to.StringPtr(storageWebEndpointForDev),
				},
			},
		},
	}
	testOIDCKeyBitSize := 256
	uploadResponse := azblob.UploadBufferResponse{}

	for _, tt := range []struct {
		name                              string
		oc                                *api.OpenShiftClusterDocument
		mocks                             func(*mock_blob.MockManager, *mock_env.MockInterface, *mock_azblob.MockBlobsClient)
		wantedOIDCIssuer                  *api.OIDCIssuer
		wantErr                           string
		wantBoundServiceAccountSigningKey bool
	}{
		{
			name: "Success - Exit createOIDC for non MIWI clusters that has ServicePrincipalProfile",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				ID:  resourceID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							SPObjectID: fakeClusterSPObjectId,
						},
					},
				},
			},
			wantedOIDCIssuer:                  nil,
			wantBoundServiceAccountSigningKey: false,
		},
		{
			name: "Success - Exit createOIDC for non MIWI clusters that has no PlatformWorkloadIdentityProfile or ServicePrincipalProfile",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				ID:  resourceID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
					},
				},
			},
			wantedOIDCIssuer:                  nil,
			wantBoundServiceAccountSigningKey: false,
		},
		{
			name: "Success - Create and persist OIDC Issuer URL",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				ID:  clusterID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
				},
			},
			mocks: func(blob *mock_blob.MockManager, menv *mock_env.MockInterface, blobsClient *mock_azblob.MockBlobsClient) {
				menv.EXPECT().FeatureIsSet(env.FeatureRequireOIDCStorageWebEndpoint).Return(false)
				menv.EXPECT().OIDCKeyBitSize().Return(testOIDCKeyBitSize)
				menv.EXPECT().OIDCEndpoint().Return(afdEndpoint)
				menv.EXPECT().OIDCStorageAccountName().Return(oidcStorageAccountName)
				menv.EXPECT().Environment().Return(&azureclient.PublicCloud)
				blob.EXPECT().GetBlobsClient(blobContainerURL).Return(blobsClient, nil)
				blobsClient.EXPECT().UploadBuffer(gomock.Any(), "", oidcbuilder.DocumentKey(oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, clusterID), oidcbuilder.DiscoveryDocumentKey), gomock.Any(), nil).Return(uploadResponse, nil)
				blobsClient.EXPECT().UploadBuffer(gomock.Any(), "", oidcbuilder.DocumentKey(oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, clusterID), oidcbuilder.JWKSKey), gomock.Any(), nil).Return(uploadResponse, nil)
			},
			wantedOIDCIssuer:                  pointerutils.ToPtr(api.OIDCIssuer(prodOIDCIssuer)),
			wantBoundServiceAccountSigningKey: true,
		},
		{
			name: "Success - Create and persist OIDC Issuer URL - Dev Env",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				ID:  clusterID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
				},
			},
			mocks: func(blob *mock_blob.MockManager, menv *mock_env.MockInterface, blobsClient *mock_azblob.MockBlobsClient) {
				menv.EXPECT().FeatureIsSet(env.FeatureRequireOIDCStorageWebEndpoint).Return(true)
				menv.EXPECT().ResourceGroup().Return(resourceGroupName)
				menv.EXPECT().OIDCKeyBitSize().Return(testOIDCKeyBitSize)
				menv.EXPECT().OIDCStorageAccountName().AnyTimes().Return(oidcStorageAccountName)
				blob.EXPECT().GetContainerProperties(gomock.Any(), resourceGroupName, oidcStorageAccountName, oidcbuilder.WebContainer).Return(containerProperties, nil)
				menv.EXPECT().Environment().Return(&azureclient.PublicCloud)
				blob.EXPECT().GetBlobsClient(blobContainerURL).Return(blobsClient, nil)
				blobsClient.EXPECT().UploadBuffer(gomock.Any(), "", oidcbuilder.DocumentKey(oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, clusterID), oidcbuilder.DiscoveryDocumentKey), gomock.Any(), nil).Return(uploadResponse, nil)
				blobsClient.EXPECT().UploadBuffer(gomock.Any(), "", oidcbuilder.DocumentKey(oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, clusterID), oidcbuilder.JWKSKey), gomock.Any(), nil).Return(uploadResponse, nil)
			},
			wantedOIDCIssuer:                  pointerutils.ToPtr(api.OIDCIssuer(devOIDCIssuer)),
			wantBoundServiceAccountSigningKey: true,
		},
		{
			name: "Fail - Get Container Properties throws error",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				ID:  clusterID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
				},
			},
			mocks: func(blob *mock_blob.MockManager, menv *mock_env.MockInterface, azblob *mock_azblob.MockBlobsClient) {
				menv.EXPECT().FeatureIsSet(env.FeatureRequireOIDCStorageWebEndpoint).Return(true)
				menv.EXPECT().ResourceGroup().Return(resourceGroupName)
				menv.EXPECT().OIDCStorageAccountName().AnyTimes().Return(oidcStorageAccountName)
				blob.EXPECT().GetContainerProperties(gomock.Any(), resourceGroupName, oidcStorageAccountName, oidcbuilder.WebContainer).Return(containerProperties, errors.New("generic error"))
			},
			wantBoundServiceAccountSigningKey: false,
			wantErr:                           "generic error",
		},
		{
			name: "Fail - blobsClient creation failure",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				ID:  clusterID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
				},
			},
			mocks: func(blob *mock_blob.MockManager, menv *mock_env.MockInterface, blobsClient *mock_azblob.MockBlobsClient) {
				menv.EXPECT().FeatureIsSet(env.FeatureRequireOIDCStorageWebEndpoint).Return(false)
				menv.EXPECT().OIDCKeyBitSize().Return(testOIDCKeyBitSize)
				menv.EXPECT().OIDCEndpoint().Return(afdEndpoint)
				menv.EXPECT().OIDCStorageAccountName().Return(oidcStorageAccountName)
				menv.EXPECT().Environment().Return(&azureclient.PublicCloud)
				blob.EXPECT().GetBlobsClient(blobContainerURL).Return(blobsClient, errors.New("generic error"))
			},
			wantBoundServiceAccountSigningKey: false,
			wantErr:                           "generic error",
		},
		{
			name: "Fail - OIDCBuilder fails to populate OIDC blob",
			oc: &api.OpenShiftClusterDocument{
				Key: strings.ToLower(resourceID),
				ID:  clusterID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
				},
			},
			mocks: func(blob *mock_blob.MockManager, menv *mock_env.MockInterface, blobsClient *mock_azblob.MockBlobsClient) {
				menv.EXPECT().FeatureIsSet(env.FeatureRequireOIDCStorageWebEndpoint).Return(false)
				menv.EXPECT().OIDCKeyBitSize().Return(testOIDCKeyBitSize)
				menv.EXPECT().OIDCEndpoint().Return(afdEndpoint)
				menv.EXPECT().OIDCStorageAccountName().Return(oidcStorageAccountName)
				menv.EXPECT().Environment().Return(&azureclient.PublicCloud)
				blob.EXPECT().GetBlobsClient(blobContainerURL).Return(blobsClient, nil)
				blobsClient.EXPECT().UploadBuffer(gomock.Any(), "", oidcbuilder.DocumentKey(oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, clusterID), oidcbuilder.DiscoveryDocumentKey), gomock.Any(), nil).Return(uploadResponse, errors.New("generic error"))
			},
			wantBoundServiceAccountSigningKey: false,
			wantErr:                           "generic error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			dbOpenShiftClusters, _ := testdatabase.NewFakeOpenShiftClusters()

			rpBlobManager := mock_blob.NewMockManager(controller)
			env := mock_env.NewMockInterface(controller)
			blobsClient := mock_azblob.NewMockBlobsClient(controller)
			if tt.mocks != nil {
				tt.mocks(rpBlobManager, env, blobsClient)
			}

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
				db:              dbOpenShiftClusters,
				log:             logrus.NewEntry(logrus.StandardLogger()),
				rpBlob:          rpBlobManager,
				doc:             doc,
				env:             env,
				subscriptionDoc: m.subscriptionDoc,
			}

			err = m.createOIDC(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			checkDoc, err := dbOpenShiftClusters.Get(ctx, strings.ToLower(resourceID))
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantedOIDCIssuer == nil && checkDoc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer != nil {
				t.Fatalf("Expected OIDC Issuer URL as nil but got a value as %s", *checkDoc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer)
			}

			if tt.wantedOIDCIssuer != nil && checkDoc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer == nil {
				t.Fatalf("OIDC Issuer URL isn't as expected, wanted %s returned nil", *tt.wantedOIDCIssuer)
			}

			if tt.wantedOIDCIssuer != nil && checkDoc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer == nil && *tt.wantedOIDCIssuer != *checkDoc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer {
				t.Fatalf("OIDC Issuer URL isn't as expected, wanted %s returned %s", *tt.wantedOIDCIssuer, *checkDoc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer)
			}

			if checkDoc.OpenShiftCluster.Properties.ClusterProfile.BoundServiceAccountSigningKey == nil && tt.wantBoundServiceAccountSigningKey {
				t.Fatalf("Bound Service Account Token is not as expected - wantBoundServiceAccountSigningKey is %t", tt.wantBoundServiceAccountSigningKey)
			}
		})
	}
}

func TestGenerateFederatedIdentityCredentials(t *testing.T) {
	ctx := context.Background()
	docID := "00000000-0000-0000-0000-000000000000"
	subID := "00000000-0000-0000-0000-000000000000"
	afdEndpoint := "fake.oic.aro.test.net"
	clusterResourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/fakeResourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/fakeCluster"
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/fakeResourceGroup/providers/Microsoft.ManagedIdentity/userAssignedIdentities"
	tenantId := "00000000-0000-0000-0000-000000000000"
	OIDCIssuer := pointerutils.ToPtr(api.OIDCIssuer(fmt.Sprintf("https://%s/%s", afdEndpoint, oidcbuilder.GetBlobName(tenantId, docID))))
	fakeClint, _ := utilmsi.NewTestFederatedIdentityCredentialsClient(subID)

	for _, tt := range []struct {
		name    string
		oc      *api.OpenShiftClusterDocument
		fixture func(f *testdatabase.Fixture)
		wantErr string
	}{
		{
			name: "Success - Exit generateFederatedIdentityCredentials for non MIWI clusters that has ServicePrincipalProfile",
			oc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: resourceGroup,
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							SPObjectID: fakeClusterSPObjectId,
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "Success - Exit generateFederatedIdentityCredentials for non MIWI clusters that has no PlatformWorkloadIdentityProfile",
			oc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:         clusterResourceID,
					Properties: api.OpenShiftClusterProperties{},
				},
			},
			wantErr: "",
		},
		{
			name: "Success - Generate Federated Identity Credentials for MIWI cluster",
			oc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							OIDCIssuer: OIDCIssuer,
							Version:    "4.14.40",
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", resourceID, "ccm"),
								},
								"ClusterIngressOperator": {
									ResourceID: fmt.Sprintf("%s/%s", resourceID, "cio"),
								},
							},
						},
					},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "testRoleSet",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{
									OperatorName:    "CloudControllerManager",
									ServiceAccounts: []string{"openshift-cloud-controller-manager:cloud-controller-manager"},
								},
								{
									OperatorName:    "ClusterIngressOperator",
									ServiceAccounts: []string{"openshift-ingress-operator:ingress-operator"},
								},
							},
						},
					},
				},
					&api.PlatformWorkloadIdentityRoleSetDocument{
						PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
							Name: "testRoleSet",
							Properties: api.PlatformWorkloadIdentityRoleSetProperties{
								OpenShiftVersion: "4.15",
								PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
									{
										OperatorName:    "CloudControllerManager",
										ServiceAccounts: []string{"openshift-cloud-controller-manager:cloud-controller-manager"},
									},
									{
										OperatorName:    "ClusterIngressOperator",
										ServiceAccounts: []string{"openshift-ingress-operator:ingress-operator"},
									},
								},
							},
						},
					},
				)
			},
			wantErr: "",
		},
		{
			name: "Fail - OIDCIssuer is nil",
			oc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"CloudControllerManager": {
									ResourceID: fmt.Sprintf("%s/%s", resourceID, "ccm"),
								},
								"ClusterIngressOperator": {
									ResourceID: fmt.Sprintf("%s/%s", resourceID, "cio"),
								},
							},
						},
					},
				},
			},
			wantErr: "OIDCIssuer is nil",
		},
		{
			name: "Success - Operator name do not exists in PlatformWorkloadIdentityProfile",
			oc: &api.OpenShiftClusterDocument{
				ID: docID,
				OpenShiftCluster: &api.OpenShiftCluster{
					ID: clusterResourceID,
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							OIDCIssuer: OIDCIssuer,
							Version:    "4.14.40",
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							UpgradeableTo: ptr.To(api.UpgradeableTo("4.15.40")),
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"DummyOperator": {
									ResourceID: fmt.Sprintf("%s/%s", resourceID, "ccm"),
								},
							},
						},
					},
				},
			},
			fixture: func(f *testdatabase.Fixture) {
				f.AddPlatformWorkloadIdentityRoleSetDocuments(&api.PlatformWorkloadIdentityRoleSetDocument{
					PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
						Name: "CloudControllerManager",
						Properties: api.PlatformWorkloadIdentityRoleSetProperties{
							OpenShiftVersion: "4.14",
							PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
								{OperatorName: "CloudControllerManager"},
							},
						},
					},
				},
					&api.PlatformWorkloadIdentityRoleSetDocument{
						PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{
							Name: "CloudControllerManager",
							Properties: api.PlatformWorkloadIdentityRoleSetProperties{
								OpenShiftVersion: "4.15",
								PlatformWorkloadIdentityRoles: []api.PlatformWorkloadIdentityRole{
									{OperatorName: "CloudControllerManager"},
								},
							},
						},
					},
				)
			},
			wantErr: "",
		},
	} {
		uuidGen := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.OPENSHIFT_VERSIONS)
		dbPlatformWorkloadIdentityRoleSets, _ := testdatabase.NewFakePlatformWorkloadIdentityRoleSets(uuidGen)
		f := testdatabase.NewFixture().WithPlatformWorkloadIdentityRoleSets(dbPlatformWorkloadIdentityRoleSets, uuidGen)
		pir := platformworkloadidentity.NewPlatformWorkloadIdentityRolesByVersionService()

		if tt.fixture != nil {
			tt.fixture(f)
			err := f.Create()
			if err != nil {
				t.Fatal(err)
			}

			err = pir.PopulatePlatformWorkloadIdentityRolesByVersion(ctx, tt.oc.OpenShiftCluster, dbPlatformWorkloadIdentityRoleSets)
			if err != nil {
				t.Fatal(err)
			}
		}

		t.Run(tt.name, func(t *testing.T) {
			m := &manager{
				log:                                    logrus.NewEntry(logrus.StandardLogger()),
				doc:                                    tt.oc,
				platformWorkloadIdentityRolesByVersion: pir,
				clusterMsiFederatedIdentityCredentials: fakeClint,
			}

			err := m.federateIdentityCredentials(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
