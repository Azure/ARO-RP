package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_storage "github.com/Azure/ARO-RP/pkg/util/mocks/storage"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
)

func TestMigrateStorageAccounts(t *testing.T) {
	ctx := context.Background()
	location := "eastus"

	rpSubscriptionId := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	rpResourceGroup := "aro-" + location
	gwyResourceGroup := "aro-gwy-" + location

	wantVnetRuleRpPrivateEndpoint := mgmtstorage.VirtualNetworkRule{
		VirtualNetworkResourceID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet", rpSubscriptionId, rpResourceGroup)),
		Action:                   mgmtstorage.Allow,
	}
	wantVnetRuleRpSubnet := mgmtstorage.VirtualNetworkRule{
		VirtualNetworkResourceID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/rp-vnet/subnets/rp-subnet", rpSubscriptionId, rpResourceGroup)),
		Action:                   mgmtstorage.Allow,
	}
	wantVnetRuleHive := mgmtstorage.VirtualNetworkRule{
		VirtualNetworkResourceID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/aks-net/subnets/PodSubnet-001", rpSubscriptionId, rpResourceGroup)),
		Action:                   mgmtstorage.Allow,
	}
	wantVnetRuleGateway := mgmtstorage.VirtualNetworkRule{
		VirtualNetworkResourceID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/gateway-vnet/subnets/gateway-subnet", rpSubscriptionId, gwyResourceGroup)),
		Action:                   mgmtstorage.Allow,
	}

	clusterResourceGroup := "testgroup"
	clusterResourceGroupId := fmt.Sprintf("/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/%s", clusterResourceGroup)
	storageSuffix := "asdfg"
	imageRegistryStorageAccountName := "imageregistryasdfg"

	masterSubnetId := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-00000000000/resourceGroups/%s/providers/Microsoft.network/virtualNetworks/aro-vnet/subnets/master-subnet", clusterResourceGroup)
	workerSubnetId := fmt.Sprintf("/subscriptions/00000000-0000-0000-0000-00000000000/resourceGroups/%s/providers/Microsoft.network/virtualNetworks/aro-vnet/subnets/master-subnet", clusterResourceGroup)

	for _, tt := range []struct {
		name                      string
		enableServiceEndpoints    bool
		wantClusterSAUpdate       mgmtstorage.AccountUpdateParameters
		wantImageRegistrySAUpdate mgmtstorage.AccountUpdateParameters
	}{
		{
			name: "default cluster works as expected",
			wantClusterSAUpdate: mgmtstorage.AccountUpdateParameters{
				AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
					AllowBlobPublicAccess:  to.BoolPtr(false),
					EnableHTTPSTrafficOnly: to.BoolPtr(true),
					MinimumTLSVersion:      mgmtstorage.TLS12,
					NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
						Bypass: mgmtstorage.AzureServices,
						VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{
							wantVnetRuleRpPrivateEndpoint,
							wantVnetRuleRpSubnet,
							wantVnetRuleHive,
							wantVnetRuleGateway,
						},
						DefaultAction: mgmtstorage.DefaultActionDeny,
					},
				},
			},
			wantImageRegistrySAUpdate: mgmtstorage.AccountUpdateParameters{
				AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
					AllowBlobPublicAccess:  to.BoolPtr(false),
					EnableHTTPSTrafficOnly: to.BoolPtr(true),
					MinimumTLSVersion:      mgmtstorage.TLS12,
					NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
						Bypass: mgmtstorage.AzureServices,
						VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{
							wantVnetRuleRpPrivateEndpoint,
							wantVnetRuleRpSubnet,
							wantVnetRuleGateway,
						},
						DefaultAction: mgmtstorage.DefaultActionDeny,
					},
				},
			},
		},
		{
			name:                   "cluster subnets with service endpoints are added to storage account vnet rules",
			enableServiceEndpoints: true,
			wantClusterSAUpdate: mgmtstorage.AccountUpdateParameters{
				AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
					AllowBlobPublicAccess:  to.BoolPtr(false),
					EnableHTTPSTrafficOnly: to.BoolPtr(true),
					MinimumTLSVersion:      mgmtstorage.TLS12,
					NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
						Bypass: mgmtstorage.AzureServices,
						VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{
							wantVnetRuleRpPrivateEndpoint,
							wantVnetRuleRpSubnet,
							{
								VirtualNetworkResourceID: to.StringPtr(strings.ToLower(masterSubnetId)),
								Action:                   mgmtstorage.Allow,
							},
							{
								VirtualNetworkResourceID: to.StringPtr(strings.ToLower(workerSubnetId)),
								Action:                   mgmtstorage.Allow,
							},
							wantVnetRuleHive,
							wantVnetRuleGateway,
						},
						DefaultAction: mgmtstorage.DefaultActionDeny,
					},
				},
			},
			wantImageRegistrySAUpdate: mgmtstorage.AccountUpdateParameters{
				AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
					AllowBlobPublicAccess:  to.BoolPtr(false),
					EnableHTTPSTrafficOnly: to.BoolPtr(true),
					MinimumTLSVersion:      mgmtstorage.TLS12,
					NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
						Bypass: mgmtstorage.AzureServices,
						VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{
							wantVnetRuleRpPrivateEndpoint,
							wantVnetRuleRpSubnet,
							{
								VirtualNetworkResourceID: to.StringPtr(strings.ToLower(masterSubnetId)),
								Action:                   mgmtstorage.Allow,
							},
							{
								VirtualNetworkResourceID: to.StringPtr(strings.ToLower(workerSubnetId)),
								Action:                   mgmtstorage.Allow,
							},
							wantVnetRuleGateway,
						},
						DefaultAction: mgmtstorage.DefaultActionDeny,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)
			subnet := mock_subnet.NewMockManager(controller)
			storage := mock_storage.NewMockManager(controller)

			env.EXPECT().Location().AnyTimes().Return(location)
			env.EXPECT().SubscriptionID().AnyTimes().Return(rpSubscriptionId)
			env.EXPECT().ResourceGroup().AnyTimes().Return(rpResourceGroup)
			env.EXPECT().GatewayResourceGroup().AnyTimes().Return(gwyResourceGroup)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)

			masterSubnet := &mgmtnetwork.Subnet{
				SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{},
				},
			}
			workerSubnet := &mgmtnetwork.Subnet{
				SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
					ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{},
				},
			}
			if tt.enableServiceEndpoints {
				serviceEndpoint := mgmtnetwork.ServiceEndpointPropertiesFormat{
					Service:   to.StringPtr(storageServiceEndpoint),
					Locations: &[]string{location},
				}
				masterSubnet.SubnetPropertiesFormat.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{serviceEndpoint}
				workerSubnet.SubnetPropertiesFormat.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{serviceEndpoint}
			}

			subnet.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(masterSubnetId)).Return(masterSubnet, nil)
			subnet.EXPECT().Get(gomock.Eq(ctx), gomock.Eq(strings.ToLower(workerSubnetId))).Return(workerSubnet, nil)

			var gotClusterSAUpdate, gotImageRegistrySAUpdate mgmtstorage.AccountUpdateParameters

			storage.EXPECT().UpdateAccount(gomock.Eq(ctx), gomock.Eq(clusterResourceGroup), gomock.Any(), gomock.Any()).
				AnyTimes().
				DoAndReturn(func(ctx context.Context, resourceGroup, accountName string, parameters mgmtstorage.AccountUpdateParameters) (mgmtstorage.Account, error) {
					switch accountName {
					case "cluster" + storageSuffix:
						gotClusterSAUpdate = parameters
					case imageRegistryStorageAccountName:
						gotImageRegistrySAUpdate = parameters
					}
					return mgmtstorage.Account{}, nil
				})

			m := &manager{
				log: logrus.NewEntry(logrus.StandardLogger()),

				env:     env,
				subnet:  subnet,
				storage: storage,

				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: clusterResourceGroupId,
							},
							MasterProfile: api.MasterProfile{
								SubnetID: masterSubnetId,
							},
							WorkerProfiles: []api.WorkerProfile{
								{
									SubnetID: workerSubnetId,
								},
							},
							StorageSuffix:                   storageSuffix,
							ImageRegistryStorageAccountName: imageRegistryStorageAccountName,
						},
					},
				},
				installViaHive: true,
			}

			err := m.migrateStorageAccounts(ctx)

			if err != nil {
				t.Errorf("expected no error but got %v", err)
			}
			if diff := cmp.Diff(gotClusterSAUpdate, tt.wantClusterSAUpdate, gocmp.Comparer(resourceIdComparer)); diff != "" {
				t.Error(diff)
			}
			if diff := cmp.Diff(gotImageRegistrySAUpdate, tt.wantImageRegistrySAUpdate, gocmp.Comparer(resourceIdComparer)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func resourceIdComparer(x, y string) bool {
	if strings.HasPrefix(x, "/subscriptions/") {
		return strings.EqualFold(x, y)
	}
	return x == y
}
