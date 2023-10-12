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

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	utilrand "k8s.io/apimachinery/pkg/util/rand"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_features "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/features"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
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
