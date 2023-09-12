package nsg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	//	"context"
	//	"errors"
	"fmt"
	"net/http"
	//	"testing"
	//	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	//	"github.com/golang/mock/gomock"
	//	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	// mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	// mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	// utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	nsgRuleName1 = "RuleName1"
	nsgRuleName2 = "RuleName2"
	nsgRuleName3 = "RuleName3"

	priority1 int32 = 201
	priority2 int32 = 202
	priority3 int32 = 203

	notOverlappingCIDR1 = "192.168.0.0/24"
	notOverlappingCIDR2 = "172.28.0.0/24"

	subsetOfMaster1 = "10.0.0.2"
	subsetOfMaster2 = "10.0.0.3/32"

	notOverlappingPrefixes1 = []string{"11.0.0.0/24", "11.0.1.0/24"}
	notOverlappingPrefixes2 = []string{"12.0.0.0/24", "12.0.1.0/24"}

	overlappingWorkerPrefixes = []string{"10.0.1.1/32", "10.0.1.2"}

	subscriptionID    = "0000-0000-0000-0000"
	resourcegroupName = "myRG"
	vNetName          = "myVnet"

	masterSubnetName = "mastersubnet"
	masterRange      = "10.0.0.0/24"
	masterSubnetID   = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", subscriptionID, resourcegroupName, vNetName, masterSubnetName)

	workerSubnet1Name = "wsubnet1"
	worker1Range      = "10.0.1.0/24"
	workerSubnet1ID   = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", subscriptionID, resourcegroupName, vNetName, workerSubnet1Name)

	workerSubnet2Name = "wsubnet2"
	worker2Range      = "10.0.2.0/24"
	workerSubnet2ID   = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s", subscriptionID, resourcegroupName, vNetName, workerSubnet2Name)

	masterSubnetMetricDimensions = map[string]string{
		dimension.ResourceID:     ocID,
		dimension.Location:       ocLocation,
		dimension.Subnet:         masterSubnetName,
		dimension.Vnet:           vNetName,
		dimension.ResourceGroup:  resourcegroupName,
		dimension.SubscriptionID: subscriptionID,
	}

	workerSubnet1MetricDimensions = map[string]string{
		dimension.ResourceID:     ocID,
		dimension.Location:       ocLocation,
		dimension.Subnet:         workerSubnet1Name,
		dimension.Vnet:           vNetName,
		dimension.ResourceGroup:  resourcegroupName,
		dimension.SubscriptionID: subscriptionID,
	}

	workerSubnet2MetricDimensions = map[string]string{
		dimension.ResourceID:     ocID,
		dimension.Location:       ocLocation,
		dimension.Subnet:         workerSubnet2Name,
		dimension.Vnet:           vNetName,
		dimension.ResourceGroup:  resourcegroupName,
		dimension.SubscriptionID: subscriptionID,
	}

	ocClusterName = "testing"
	ocID          = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/%s", subscriptionID, resourcegroupName, ocClusterName)
	ocLocation    = "eastus"
	oc            = api.OpenShiftCluster{
		ID:       ocID,
		Location: ocLocation,
		Properties: api.OpenShiftClusterProperties{
			MasterProfile: api.MasterProfile{
				SubnetID: masterSubnetID,
			},
			WorkerProfiles: []api.WorkerProfile{
				{
					SubnetID: workerSubnet1ID,
				},
				{
					SubnetID: workerSubnet2ID,
				},
			},
		},
	}

	nsg1Name = "NSG-1"
	nsg1ID   = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s", subscriptionID, resourcegroupName, nsg1Name)
	nsg2Name = "NSG-2"
	nsg2ID   = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s", subscriptionID, resourcegroupName, nsg2Name)
)

func createBaseSubnets() (network.Subnet, network.Subnet, network.Subnet) {
	subnets := make([]network.Subnet, 0, 3)
	ranges := []string{masterRange, worker1Range, worker2Range}

	for i := 0; i < 3; i++ {
		subnets = append(
			subnets,
			network.Subnet{
				Response: autorest.Response{
					Response: &http.Response{
						StatusCode: 200,
					},
				},
				SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
					AddressPrefix: &ranges[0],
				},
			},
		)
	}
	return subnets[0], subnets[1], subnets[2]
}

//func TestMonitor(t *testing.T) {
//	ctx := context.Background()
//
//	for _, tt := range []struct {
//		name        string
//		mockSubnet  func(*mock_network.MockSubnetsClient)
//		mockEmitter func(*mock_metrics.MockEmitter)
//		wantErr     string
//	}{
//		{
//			name: "fail - forbidden access when retrieving worker subnet 2",
//			mockSubnet: func(mock *mock_network.MockSubnetsClient) {
//				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
//				workerSubnet2.StatusCode = 403
//
//				_1 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, masterSubnetName, expandNSG).
//					Return(masterSubnet, nil)
//
//				_2 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, expandNSG).
//					Return(workerSubnet1, nil)
//
//				_3 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, expandNSG).
//					Return(workerSubnet2, errors.New("Error while retrieving worker subnet 2"))
//
//				gomock.InOrder(_1, _2, _3)
//			},
//			mockEmitter: func(mock *mock_metrics.MockEmitter) {
//				_1 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), masterSubnetMetricDimensions)
//				_2 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet1MetricDimensions)
//				_3 := mock.EXPECT().EmitGauge(MetricSubnetAccessForbidden, int64(1), workerSubnet2MetricDimensions)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//			wantErr: "Error while retrieving worker subnet 2",
//		},
//		{
//			name: "pass - no Deny NSG rules",
//			mockSubnet: func(mock *mock_network.MockSubnetsClient) {
//				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
//				nsg := network.SecurityGroup{
//					ID: &nsg1ID,
//					SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
//						SecurityRules: &[]network.SecurityRule{
//							{
//								Name: &nsgRuleName1,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access: network.SecurityRuleAccessAllow,
//								},
//							},
//							{
//								Name: &nsgRuleName2,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access: network.SecurityRuleAccessAllow,
//								},
//							},
//						},
//					},
//				}
//				masterSubnet.NetworkSecurityGroup = &nsg
//				workerSubnet1.NetworkSecurityGroup = &nsg
//				workerSubnet2.NetworkSecurityGroup = &nsg
//
//				_1 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, masterSubnetName, expandNSG).
//					Return(masterSubnet, nil)
//
//				_2 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, expandNSG).
//					Return(workerSubnet1, nil)
//
//				_3 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, expandNSG).
//					Return(workerSubnet2, nil)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//			mockEmitter: func(mock *mock_metrics.MockEmitter) {
//				_1 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), masterSubnetMetricDimensions)
//				_2 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet1MetricDimensions)
//				_3 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet2MetricDimensions)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//		},
//		{
//			name: "pass - no rules",
//			mockSubnet: func(mock *mock_network.MockSubnetsClient) {
//				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
//				nsg := network.SecurityGroup{
//					ID: &nsg1ID,
//					SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
//						SecurityRules: &[]network.SecurityRule{},
//					},
//				}
//				masterSubnet.NetworkSecurityGroup = &nsg
//				workerSubnet1.NetworkSecurityGroup = &nsg
//				workerSubnet2.NetworkSecurityGroup = &nsg
//
//				_1 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, masterSubnetName, expandNSG).
//					Return(masterSubnet, nil)
//
//				_2 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, expandNSG).
//					Return(workerSubnet1, nil)
//
//				_3 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, expandNSG).
//					Return(workerSubnet2, nil)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//			mockEmitter: func(mock *mock_metrics.MockEmitter) {
//				_1 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), masterSubnetMetricDimensions)
//				_2 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet1MetricDimensions)
//				_3 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet2MetricDimensions)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//		},
//		{
//			name: "pass - only irrelevant deny rules",
//			mockSubnet: func(mock *mock_network.MockSubnetsClient) {
//				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
//				masterSubnet.AddressPrefix = &masterRange
//				nsg1 := network.SecurityGroup{
//					ID: &nsg1ID,
//					SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
//						SecurityRules: &[]network.SecurityRule{
//							{
//								Name: &nsgRuleName1,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access:                   network.SecurityRuleAccessDeny,
//									SourceAddressPrefix:      &notOverlappingCIDR1,
//									DestinationAddressPrefix: &notOverlappingCIDR2,
//								},
//							},
//							{
//								Name: &nsgRuleName1,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access:                     network.SecurityRuleAccessDeny,
//									SourceAddressPrefixes:      &notOverlappingPrefixes1,
//									DestinationAddressPrefixes: &notOverlappingPrefixes2,
//								},
//							},
//						},
//					},
//				}
//				nsg2 := network.SecurityGroup{
//					ID: &nsg2ID,
//					SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
//						SecurityRules: &[]network.SecurityRule{
//							{
//								Name: &nsgRuleName1,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access:                   network.SecurityRuleAccessDeny,
//									SourceAddressPrefix:      &notOverlappingCIDR1,
//									DestinationAddressPrefix: &notOverlappingCIDR2,
//								},
//							},
//							{
//								Name: &nsgRuleName1,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access:                     network.SecurityRuleAccessDeny,
//									SourceAddressPrefixes:      &notOverlappingPrefixes1,
//									DestinationAddressPrefixes: &notOverlappingPrefixes2,
//								},
//							},
//						},
//					},
//				}
//				masterSubnet.NetworkSecurityGroup = &nsg1
//				workerSubnet1.NetworkSecurityGroup = &nsg2
//				workerSubnet2.NetworkSecurityGroup = &nsg2
//
//				_1 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, masterSubnetName, expandNSG).
//					Return(masterSubnet, nil)
//
//				_2 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, expandNSG).
//					Return(workerSubnet1, nil)
//
//				_3 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, expandNSG).
//					Return(workerSubnet2, nil)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//			mockEmitter: func(mock *mock_metrics.MockEmitter) {
//				_1 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), masterSubnetMetricDimensions)
//				_2 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet1MetricDimensions)
//				_3 := mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet2MetricDimensions)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//		},
//		{
//			name: "pass - invalid deny rules, emitting metrics",
//			mockSubnet: func(mock *mock_network.MockSubnetsClient) {
//				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
//				masterSubnet.AddressPrefix = &masterRange
//				workerSubnet1.AddressPrefix = &worker1Range
//				workerSubnet2.AddressPrefix = &worker2Range
//				nsg1 := network.SecurityGroup{
//					ID: &nsg1ID,
//					SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
//						SecurityRules: &[]network.SecurityRule{
//							{
//								Name: &nsgRuleName1,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access:                   network.SecurityRuleAccessDeny,
//									SourceAddressPrefix:      &subsetOfMaster1,
//									DestinationAddressPrefix: &subsetOfMaster2,
//									Priority:                 &priority1,
//									Direction:                network.SecurityRuleDirectionInbound,
//								},
//							},
//							{
//								Name: &nsgRuleName2,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access:                     network.SecurityRuleAccessDeny,
//									SourceAddressPrefixes:      &notOverlappingPrefixes1,
//									DestinationAddressPrefixes: &notOverlappingPrefixes2,
//									Priority:                   &priority2,
//								},
//							},
//						},
//					},
//				}
//				asterisk := "*"
//				nsg2 := network.SecurityGroup{
//					ID: &nsg2ID,
//					SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
//						SecurityRules: &[]network.SecurityRule{
//							{
//								Name: &nsgRuleName3,
//								SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//									Access:                   network.SecurityRuleAccessDeny,
//									SourceAddressPrefixes:    &overlappingWorkerPrefixes,
//									DestinationAddressPrefix: &asterisk,
//									Priority:                 &priority3,
//									Direction:                network.SecurityRuleDirectionOutbound,
//								},
//							},
//						},
//					},
//				}
//				masterSubnet.NetworkSecurityGroup = &nsg1
//				workerSubnet1.NetworkSecurityGroup = &nsg2
//				workerSubnet2.NetworkSecurityGroup = &nsg2
//
//				_1 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, masterSubnetName, expandNSG).
//					Return(masterSubnet, nil)
//
//				_2 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, expandNSG).
//					Return(workerSubnet1, nil)
//
//				_3 := mock.EXPECT().
//					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, expandNSG).
//					Return(workerSubnet2, nil)
//
//				gomock.InOrder(_1, _2, _3)
//			},
//			mockEmitter: func(mock *mock_metrics.MockEmitter) {
//				mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), masterSubnetMetricDimensions)
//				mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet1MetricDimensions)
//				mock.EXPECT().EmitGauge(MetricSubnetAccessResponseCode, int64(http.StatusOK), workerSubnet2MetricDimensions)
//				mock.EXPECT().EmitGauge(MetricInvalidDenyRule, int64(1), map[string]string{
//					dimension.ResourceID:          ocID,
//					dimension.Location:            ocLocation,
//					dimension.SubscriptionID:      subscriptionID,
//					dimension.ResourceGroup:       resourcegroupName,
//					dimension.ResourceName:        nsg1Name,
//					dimension.NSGRuleName:         nsgRuleName1,
//					dimension.NSGRuleSources:      subsetOfMaster1,
//					dimension.NSGRuleDestinations: subsetOfMaster2,
//					dimension.NSGRuleDirection:    string(network.SecurityRuleDirectionInbound),
//					dimension.NSGRulePriority:     string(priority1),
//				})
//				mock.EXPECT().EmitGauge(MetricInvalidDenyRule, int64(1), map[string]string{
//					dimension.ResourceID:          ocID,
//					dimension.Location:            ocLocation,
//					dimension.SubscriptionID:      subscriptionID,
//					dimension.ResourceGroup:       resourcegroupName,
//					dimension.ResourceName:        nsg2Name,
//					dimension.NSGRuleName:         nsgRuleName3,
//					dimension.NSGRuleSources:      "10.0.1.1/32,10.0.1.2",
//					dimension.NSGRuleDestinations: "*",
//					dimension.NSGRuleDirection:    string(network.SecurityRuleDirectionOutbound),
//					dimension.NSGRulePriority:     string(priority3),
//				})
//			},
//		},
//	} {
//		t.Run(tt.name, func(t *testing.T) {
//			ctrl := gomock.NewController(t)
//			defer ctrl.Finish()
//			subnetClient := mock_network.NewMockSubnetsClient(ctrl)
//			emitter := mock_metrics.NewMockEmitter(ctrl)
//
//			if tt.mockSubnet != nil {
//				tt.mockSubnet(subnetClient)
//			}
//			if tt.mockEmitter != nil {
//				tt.mockEmitter(emitter)
//			}
//
//			n := NSGMonitor{
//				log:       logrus.NewEntry(logrus.New()),
//				oc:        &oc,
//				subnetCli: subnetClient,
//				emitter:   emitter,
//				done:      make(chan error),
//			}
//
//			go n.Monitor(ctx)
//
//			var err error
//			select {
//			case err = <-n.Done():
//			case <-time.After(3 * time.Second):
//				t.Error("Time out executing the test.")
//			}
//
//			utilerror.AssertErrorMessage(t, err, tt.wantErr)
//		})
//	}
//}
