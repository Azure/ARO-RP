package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"testing"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestStorageAccount(t *testing.T) {
	location := "eastus"
	rpSubscriptionId := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	rpResourceGroup := "aro-" + location
	gwyResourceGroup := "aro-gwy-" + location

	// we expect these encryption properties set on every storage account we create
	wantEncryption := &mgmtstorage.Encryption{
		RequireInfrastructureEncryption: to.BoolPtr(true),
		Services: &mgmtstorage.EncryptionServices{
			Blob: &mgmtstorage.EncryptionService{
				KeyType: mgmtstorage.KeyTypeAccount,
				Enabled: to.BoolPtr(true),
			},
			File: &mgmtstorage.EncryptionService{
				KeyType: mgmtstorage.KeyTypeAccount,
				Enabled: to.BoolPtr(true),
			},
			Table: &mgmtstorage.EncryptionService{
				KeyType: mgmtstorage.KeyTypeAccount,
				Enabled: to.BoolPtr(true),
			},
			Queue: &mgmtstorage.EncryptionService{
				KeyType: mgmtstorage.KeyTypeAccount,
				Enabled: to.BoolPtr(true),
			},
		},
		KeySource: mgmtstorage.KeySourceMicrosoftStorage,
	}
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

	clusterMasterSubnet := "/subscriptions/00000000-0000-0000-0000-00000000000/resourceGroups/aro-cluster/providers/Microsoft.network/virtualNetworks/aro-vnet/subnets/master-subnet"
	clusterWorkerSubnet := "/subscriptions/00000000-0000-0000-0000-00000000000/resourceGroups/aro-cluster/providers/Microsoft.network/virtualNetworks/aro-vnet/subnets/worker-subnet"

	for _, tt := range []struct {
		name               string
		storageAccountName string
		ocpSubnets         []string
		installViaHive     bool
		localDev           bool
		want               *arm.Resource
	}{
		{
			name:               "imageregistry - no subnets",
			storageAccountName: "imageregistryasdfg",
			want: &arm.Resource{
				APIVersion: azureclient.APIVersion("Microsoft.Storage"),
				Resource: &mgmtstorage.Account{
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Name:     to.StringPtr("imageregistryasdfg"),
					Location: &location,
					Kind:     mgmtstorage.StorageV2,
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					AccountProperties: &mgmtstorage.AccountProperties{
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
						Encryption: wantEncryption,
					},
				},
			},
		},
		{
			name:               "imageregistry - cluster master/worker subnets",
			storageAccountName: "imageregistryasdfg",
			ocpSubnets:         []string{clusterMasterSubnet, clusterWorkerSubnet},
			want: &arm.Resource{
				APIVersion: azureclient.APIVersion("Microsoft.Storage"),
				Resource: &mgmtstorage.Account{
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Name:     to.StringPtr("imageregistryasdfg"),
					Location: &location,
					Kind:     mgmtstorage.StorageV2,
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					AccountProperties: &mgmtstorage.AccountProperties{
						AllowBlobPublicAccess:  to.BoolPtr(false),
						EnableHTTPSTrafficOnly: to.BoolPtr(true),
						MinimumTLSVersion:      mgmtstorage.TLS12,
						NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
							Bypass: mgmtstorage.AzureServices,
							VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{
								wantVnetRuleRpPrivateEndpoint,
								wantVnetRuleRpSubnet,
								{
									VirtualNetworkResourceID: &clusterMasterSubnet,
									Action:                   mgmtstorage.Allow,
								},
								{
									VirtualNetworkResourceID: &clusterWorkerSubnet,
									Action:                   mgmtstorage.Allow,
								},
								wantVnetRuleGateway,
							},
							DefaultAction: mgmtstorage.DefaultActionDeny,
						},
						Encryption: wantEncryption,
					},
				},
			},
		},
		{
			name:               "imageregistry - local dev",
			storageAccountName: "imageregistryasdfg",
			localDev:           true,
			want: &arm.Resource{
				APIVersion: azureclient.APIVersion("Microsoft.Storage"),
				Resource: &mgmtstorage.Account{
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Name:     to.StringPtr("imageregistryasdfg"),
					Location: &location,
					Kind:     mgmtstorage.StorageV2,
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					AccountProperties: &mgmtstorage.AccountProperties{
						AllowBlobPublicAccess:  to.BoolPtr(false),
						EnableHTTPSTrafficOnly: to.BoolPtr(true),
						MinimumTLSVersion:      mgmtstorage.TLS12,
						NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
							Bypass: mgmtstorage.AzureServices,
							VirtualNetworkRules: &[]mgmtstorage.VirtualNetworkRule{
								wantVnetRuleRpPrivateEndpoint,
								wantVnetRuleRpSubnet,
							},
							DefaultAction: mgmtstorage.DefaultActionAllow,
						},
						Encryption: wantEncryption,
					},
				},
			},
		},
		{
			name:               "cluster - no subnets",
			storageAccountName: "clusterasdfg",
			want: &arm.Resource{
				APIVersion: azureclient.APIVersion("Microsoft.Storage"),
				Resource: &mgmtstorage.Account{
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Name:     to.StringPtr("clusterasdfg"),
					Location: &location,
					Kind:     mgmtstorage.StorageV2,
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					AccountProperties: &mgmtstorage.AccountProperties{
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
						Encryption: wantEncryption,
					},
				},
			},
		},
		{
			name:               "cluster - no subnets, install via Hive",
			storageAccountName: "clusterasdfg",
			installViaHive:     true,
			want: &arm.Resource{
				APIVersion: azureclient.APIVersion("Microsoft.Storage"),
				Resource: &mgmtstorage.Account{
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					Name:     to.StringPtr("clusterasdfg"),
					Location: &location,
					Kind:     mgmtstorage.StorageV2,
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					AccountProperties: &mgmtstorage.AccountProperties{
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
						Encryption: wantEncryption,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			env := mock_env.NewMockInterface(controller)

			env.EXPECT().Location().AnyTimes().Return(location)
			env.EXPECT().SubscriptionID().AnyTimes().Return(rpSubscriptionId)
			env.EXPECT().ResourceGroup().AnyTimes().Return(rpResourceGroup)
			env.EXPECT().GatewayResourceGroup().AnyTimes().Return(gwyResourceGroup)
			env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(tt.localDev)

			m := &manager{
				log:            logrus.NewEntry(logrus.StandardLogger()),
				installViaHive: tt.installViaHive,
				env:            env,
			}

			got := m.storageAccount(tt.storageAccountName, location, tt.ocpSubnets)

			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}
