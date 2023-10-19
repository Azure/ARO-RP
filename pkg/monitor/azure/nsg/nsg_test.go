package nsg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
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

	notOverlappingPrefix1_1 = "11.0.0.0/24"
	notOverlappingPrefix1_2 = "11.0.1.0/24"
	notOverlappingPrefixes1 = []*string{&notOverlappingPrefix1_1, &notOverlappingPrefix1_2}
	notOverlappingPrefix2_1 = "12.0.0.0/24"
	notOverlappingPrefix2_2 = "12.0.1.0/24"
	notOverlappingPrefixes2 = []*string{&notOverlappingPrefix2_1, &notOverlappingPrefix2_2}

	overlappingWorkerPrefix1  = "10.0.1.1/32"
	overlappingWorkerPrefix2  = "10.0.1.2"
	overlappingWorkerPrefixes = []*string{&overlappingWorkerPrefix1, &overlappingWorkerPrefix2}

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

	workerSubnet2MetricDimensions = map[string]string{
		dimension.ResourceID:       ocID,
		dimension.Location:         ocLocation,
		dimension.Subnet:           workerSubnet2Name,
		dimension.Vnet:             vNetName,
		dimension.NSGResourceGroup: resourcegroupName,
		dimension.SubscriptionID:   subscriptionID,
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
					SubnetID: "", // This should still work. Customers can create a faulty MachineSet.
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

	nsgAllow    = armnetwork.SecurityRuleAccessAllow
	nsgDeny     = armnetwork.SecurityRuleAccessDeny
	nsgInbound  = armnetwork.SecurityRuleDirectionInbound
	nsgOutbound = armnetwork.SecurityRuleDirectionOutbound
)

func createBaseSubnets() (armnetwork.SubnetsClientGetResponse, armnetwork.SubnetsClientGetResponse, armnetwork.SubnetsClientGetResponse) {
	resp := make([]armnetwork.SubnetsClientGetResponse, 0, 3)
	ranges := []string{masterRange, worker1Range, worker2Range}
	// to support subnets with multiple cidrs
	multiplePrefixes := [][]string{
		{
			"10.128.0.0/24",
			"10.128.1.0/24",
		},
		{
			"10.128.2.0/24",
			"10.128.3.0/24",
		},
		{
			"10.128.4.0/24",
			"10.128.5.0/24",
		},
	}

	// even somethingn nonsense should still work
	gibberish := "JUNK"
	for i := 0; i < 3; i++ {
		resp = append(
			resp,
			armnetwork.SubnetsClientGetResponse{
				Subnet: armnetwork.Subnet{
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: &ranges[i],
						AddressPrefixes: []*string{
							&multiplePrefixes[i][0],
							&multiplePrefixes[i][1],
							&gibberish,
						},
					},
				},
			},
		)
	}
	return resp[0], resp[1], resp[2]
}

func TestMonitor(t *testing.T) {
	ctx := context.Background()
	options := &armnetwork.SubnetsClientGetOptions{
		Expand: &expandNSG,
	}

	forbiddenRespErr := azcore.ResponseError{
		StatusCode: http.StatusForbidden,
		RawResponse: &http.Response{
			Request: &http.Request{
				URL: &url.URL{},
			},
			Body: io.NopCloser(strings.NewReader("Forbidden")),
		},
	}

	for _, tt := range []struct {
		name        string
		mockSubnet  func(*mock_armnetwork.MockSubnetsClient)
		mockEmitter func(*mock_metrics.MockEmitter)
		wantErr     string
	}{
		{
			name: "fail - forbidden access when retrieving worker subnet 2",
			mockSubnet: func(mock *mock_armnetwork.MockSubnetsClient) {
				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
				_1 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, masterSubnetName, options).
					Return(masterSubnet, nil)

				_2 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, options).
					Return(workerSubnet1, nil)

				_3 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, options).
					Return(workerSubnet2, &forbiddenRespErr)

				gomock.InOrder(_1, _2, _3)
			},
			mockEmitter: func(mock *mock_metrics.MockEmitter) {
				mock.EXPECT().EmitGauge(MetricSubnetAccessForbidden, int64(1), workerSubnet2MetricDimensions)
			},
			wantErr: forbiddenRespErr.Error(),
		},
		{
			name: "pass - no Deny NSG rules",
			mockSubnet: func(mock *mock_armnetwork.MockSubnetsClient) {
				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
				nsg := armnetwork.SecurityGroup{
					ID: &nsg1ID,
					Properties: &armnetwork.SecurityGroupPropertiesFormat{
						SecurityRules: []*armnetwork.SecurityRule{
							{
								Name: &nsgRuleName1,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access: &nsgAllow,
								},
							},
							{
								Name: &nsgRuleName2,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access: &nsgAllow,
								},
							},
						},
					},
				}
				masterSubnet.Properties.NetworkSecurityGroup = &nsg

				workerSubnet1.Properties.NetworkSecurityGroup = &nsg
				workerSubnet2.Properties.NetworkSecurityGroup = &nsg

				_1 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, masterSubnetName, options).
					Return(masterSubnet, nil)

				_2 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, options).
					Return(workerSubnet1, nil)

				_3 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, options).
					Return(workerSubnet2, nil)

				gomock.InOrder(_1, _2, _3)
			},
		},
		{
			name: "pass - no rules",
			mockSubnet: func(mock *mock_armnetwork.MockSubnetsClient) {
				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
				nsg := armnetwork.SecurityGroup{
					ID: &nsg1ID,
					Properties: &armnetwork.SecurityGroupPropertiesFormat{
						SecurityRules: []*armnetwork.SecurityRule{},
					},
				}
				masterSubnet.Properties.NetworkSecurityGroup = &nsg
				workerSubnet1.Properties.NetworkSecurityGroup = &nsg
				workerSubnet2.Properties.NetworkSecurityGroup = &nsg

				_1 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, masterSubnetName, options).
					Return(masterSubnet, nil)

				_2 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, options).
					Return(workerSubnet1, nil)

				_3 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, options).
					Return(workerSubnet2, nil)

				gomock.InOrder(_1, _2, _3)
			},
		},
		{
			name: "pass - only irrelevant deny rules",
			mockSubnet: func(mock *mock_armnetwork.MockSubnetsClient) {
				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
				masterSubnet.Properties.AddressPrefix = &masterRange
				nsg1 := armnetwork.SecurityGroup{
					ID: &nsg1ID,
					Properties: &armnetwork.SecurityGroupPropertiesFormat{
						SecurityRules: []*armnetwork.SecurityRule{
							{
								Name: &nsgRuleName1,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access:                   &nsgDeny,
									SourceAddressPrefix:      &notOverlappingCIDR1,
									DestinationAddressPrefix: &notOverlappingCIDR2,
								},
							},
							{
								Name: &nsgRuleName1,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access:                     &nsgDeny,
									SourceAddressPrefixes:      notOverlappingPrefixes1,
									DestinationAddressPrefixes: notOverlappingPrefixes2,
								},
							},
						},
					},
				}
				nsg2 := armnetwork.SecurityGroup{
					ID: &nsg2ID,
					Properties: &armnetwork.SecurityGroupPropertiesFormat{
						SecurityRules: []*armnetwork.SecurityRule{
							{
								Name: &nsgRuleName1,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access:                   &nsgDeny,
									SourceAddressPrefix:      &notOverlappingCIDR1,
									DestinationAddressPrefix: &notOverlappingCIDR2,
								},
							},
							{
								Name: &nsgRuleName1,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access:                     &nsgDeny,
									SourceAddressPrefixes:      notOverlappingPrefixes1,
									DestinationAddressPrefixes: notOverlappingPrefixes2,
								},
							},
						},
					},
				}
				masterSubnet.Properties.NetworkSecurityGroup = &nsg1
				workerSubnet1.Properties.NetworkSecurityGroup = &nsg2
				workerSubnet2.Properties.NetworkSecurityGroup = &nsg2

				_1 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, masterSubnetName, options).
					Return(masterSubnet, nil)

				_2 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, options).
					Return(workerSubnet1, nil)

				_3 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, options).
					Return(workerSubnet2, nil)

				gomock.InOrder(_1, _2, _3)
			},
		},
		{
			name: "pass - invalid deny rules, emitting metrics",
			mockSubnet: func(mock *mock_armnetwork.MockSubnetsClient) {
				masterSubnet, workerSubnet1, workerSubnet2 := createBaseSubnets()
				masterSubnet.Properties.AddressPrefix = &masterRange
				workerSubnet1.Properties.AddressPrefix = &worker1Range
				workerSubnet2.Properties.AddressPrefix = &worker2Range
				nsg1 := armnetwork.SecurityGroup{
					ID: &nsg1ID,
					Properties: &armnetwork.SecurityGroupPropertiesFormat{
						SecurityRules: []*armnetwork.SecurityRule{
							{
								Name: &nsgRuleName1,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access:                   &nsgDeny,
									SourceAddressPrefix:      &subsetOfMaster1,
									DestinationAddressPrefix: &subsetOfMaster2,
									Priority:                 &priority1,
									Direction:                &nsgInbound,
								},
							},
							{
								Name: &nsgRuleName2,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access:                     &nsgDeny,
									SourceAddressPrefixes:      notOverlappingPrefixes1,
									DestinationAddressPrefixes: notOverlappingPrefixes2,
									Priority:                   &priority2,
								},
							},
						},
					},
				}
				asterisk := "*"
				nsg2 := armnetwork.SecurityGroup{
					ID: &nsg2ID,
					Properties: &armnetwork.SecurityGroupPropertiesFormat{
						SecurityRules: []*armnetwork.SecurityRule{
							{
								Name: &nsgRuleName3,
								Properties: &armnetwork.SecurityRulePropertiesFormat{
									Access:                   &nsgDeny,
									SourceAddressPrefixes:    overlappingWorkerPrefixes,
									DestinationAddressPrefix: &asterisk,
									Priority:                 &priority3,
									Direction:                &nsgOutbound,
								},
							},
						},
					},
				}
				masterSubnet.Properties.NetworkSecurityGroup = &nsg1
				workerSubnet1.Properties.NetworkSecurityGroup = &nsg2
				workerSubnet2.Properties.NetworkSecurityGroup = &nsg2

				_1 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, masterSubnetName, options).
					Return(masterSubnet, nil)

				_2 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet1Name, options).
					Return(workerSubnet1, nil)

				_3 := mock.EXPECT().
					Get(ctx, resourcegroupName, vNetName, workerSubnet2Name, options).
					Return(workerSubnet2, nil)

				gomock.InOrder(_1, _2, _3)
			},
			mockEmitter: func(mock *mock_metrics.MockEmitter) {
				mock.EXPECT().EmitGauge(MetricInvalidDenyRule, int64(1), map[string]string{
					dimension.ResourceID:          ocID,
					dimension.Location:            ocLocation,
					dimension.SubscriptionID:      subscriptionID,
					dimension.NSGResourceGroup:    resourcegroupName,
					dimension.NSG:                 nsg1Name,
					dimension.NSGRuleName:         nsgRuleName1,
					dimension.NSGRuleSources:      subsetOfMaster1,
					dimension.NSGRuleDestinations: subsetOfMaster2,
					dimension.NSGRuleDirection:    string(armnetwork.SecurityRuleDirectionInbound),
					dimension.NSGRulePriority:     fmt.Sprint(priority1),
				})
				mock.EXPECT().EmitGauge(MetricInvalidDenyRule, int64(1), map[string]string{
					dimension.ResourceID:          ocID,
					dimension.Location:            ocLocation,
					dimension.SubscriptionID:      subscriptionID,
					dimension.NSGResourceGroup:    resourcegroupName,
					dimension.NSG:                 nsg2Name,
					dimension.NSGRuleName:         nsgRuleName3,
					dimension.NSGRuleSources:      "10.0.1.1/32,10.0.1.2",
					dimension.NSGRuleDestinations: "*",
					dimension.NSGRuleDirection:    string(armnetwork.SecurityRuleDirectionOutbound),
					dimension.NSGRulePriority:     fmt.Sprint(priority3),
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			subnetClient := mock_armnetwork.NewMockSubnetsClient(ctrl)
			emitter := mock_metrics.NewMockEmitter(ctrl)

			if tt.mockSubnet != nil {
				tt.mockSubnet(subnetClient)
			}
			if tt.mockEmitter != nil {
				tt.mockEmitter(emitter)
			}

			var wg sync.WaitGroup
			n := NewNSGMonitor(logrus.NewEntry(logrus.New()), &oc, subscriptionID, subnetClient, emitter, &wg)

			wg.Add(1)
			err := n.Monitor(ctx)
			done := make(chan any)

			go wait(&wg, done)

			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Error("Timeout waiting for the monitor to finish")
			}
			if len(err) != 0 {
				utilerror.AssertErrorMessage(t, err[0], tt.wantErr)
			}
		})
	}
}

func wait(wg *sync.WaitGroup, done chan<- any) {
	wg.Wait()
	done <- nil
}
