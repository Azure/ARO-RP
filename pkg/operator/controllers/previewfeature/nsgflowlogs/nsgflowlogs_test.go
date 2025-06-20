package nsgflowlogs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

var (
	subscriptionId                  = "0000000-0000-0000-0000-000000000000"
	networkWatcherResourceGroupName = "networkWatcherRG"
	networkWatcherName              = "networkWatcher_eastus"
	networkWatcherResourceId        = "/subscriptions/" + subscriptionId + "/resourcegroups/" + networkWatcherResourceGroupName + "/providers/Microsoft.Network/networkWatchers/" + networkWatcherName
	location                        = "eastus"
	vnetResourceGroup               = "vnet-rg"
	vnetName                        = "vnet"

	subnetNameMasterNSGName = "masterNSG"
	subnetNameWorkerNSGName = "workerNSG"
	subnetNameMasterNSGID   = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/" + subnetNameMasterNSGName
	subnetNameWorkerNSGID   = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/" + subnetNameWorkerNSGName
	subnetNameWorker        = "worker"
	subnetNameWorker2       = "worker2"
	subnetNameMaster        = "master"
	resourceIdMaster        = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
	resourceIdWorker        = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
	resourceIdWorker2       = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker2
)

func getValidPreviewFeatureInstance() *aropreviewv1alpha1.PreviewFeature {
	return &aropreviewv1alpha1.PreviewFeature{
		Spec: aropreviewv1alpha1.PreviewFeatureSpec{
			NSGFlowLogs: &aropreviewv1alpha1.NSGFlowLogs{
				Enabled:                                 false,
				Version:                                 0,
				NetworkWatcherID:                        "",
				StorageAccountResourceID:                "",
				RetentionDays:                           0,
				TrafficAnalyticsLogAnalyticsWorkspaceID: "",
				TrafficAnalyticsInterval:                metav1.Duration{},
			},
		},
	}
}

func getValidFlowLogFeature() *armnetwork.FlowLog {
	return &armnetwork.FlowLog{
		Properties: &armnetwork.FlowLogPropertiesFormat{
			TargetResourceID: pointerutils.ToPtr(""),
			Enabled:          pointerutils.ToPtr(true),
			StorageID:        pointerutils.ToPtr(""),
			RetentionPolicy: &armnetwork.RetentionPolicyParameters{
				Days: pointerutils.ToPtr(int32(0)),
			},
			Format: &armnetwork.FlowLogFormatParameters{
				Type:    pointerutils.ToPtr(armnetwork.FlowLogFormatTypeJSON),
				Version: pointerutils.ToPtr(int32(0)),
			},
			FlowAnalyticsConfiguration: &armnetwork.TrafficAnalyticsProperties{
				NetworkWatcherFlowAnalyticsConfiguration: &armnetwork.TrafficAnalyticsConfigurationProperties{
					TrafficAnalyticsInterval: pointerutils.ToPtr(int32(0)),
					WorkspaceID:              pointerutils.ToPtr(""),
				},
			},
		},
		Location: pointerutils.ToPtr(location),
	}
}

func TestReconcileManager(t *testing.T) {
	for _, tt := range []struct {
		name              string
		subnetMock        func(*mock_armnetwork.MockSubnetsClient, *mock_subnet.MockKubeManager)
		instance          func(*aropreviewv1alpha1.PreviewFeature)
		flowLogClientMock func(*mock_armnetwork.MockFlowLogsClientInterface)
		wantErr           string
	}{
		{
			name: "do not enable flow log if parameters are missing/wrong",
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
					},
					{
						ResourceID: resourceIdWorker,
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameMasterNSGID,
							},
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameWorkerNSGID,
							},
						},
					},
				}, nil)
			},
			instance: func(feature *aropreviewv1alpha1.PreviewFeature) {
				feature.Spec.NSGFlowLogs.Enabled = true
			},
			wantErr: "parsing failed for . Invalid resource Id format",
		},
		{
			name: "enable flow log",
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
					},
					{
						ResourceID: resourceIdWorker,
					},
					{
						ResourceID: resourceIdWorker2,
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameMasterNSGID,
							},
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameMasterNSGID, // same NSG as the master subnet
							},
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker2, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameWorkerNSGID, // different NSG ID. expect another one call to create
							},
						},
					},
				}, nil)
			},
			flowLogClientMock: func(client *mock_armnetwork.MockFlowLogsClientInterface) {
				flowLogMaster := getValidFlowLogFeature()
				flowLogMaster.Properties.TargetResourceID = &subnetNameMasterNSGID

				flowLogWorker := getValidFlowLogFeature()
				flowLogWorker.Properties.TargetResourceID = &subnetNameWorkerNSGID
				// enable once per NSG
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameMasterNSGName, *flowLogMaster, nil)
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameWorkerNSGName, *flowLogWorker, nil)
			},
			instance: func(feature *aropreviewv1alpha1.PreviewFeature) {
				feature.Spec.NSGFlowLogs.Enabled = true
				feature.Spec.NSGFlowLogs.NetworkWatcherID = networkWatcherResourceId
			},
			wantErr: "",
		},
		{
			name: "disable flow log",
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
					},
					{
						ResourceID: resourceIdWorker,
					},
					{
						ResourceID: resourceIdWorker2,
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameMasterNSGID,
							},
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameMasterNSGID, // same NSG as the master subnet
							},
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker2, nil).Return(armnetwork.SubnetsClientGetResponse{
					Subnet: armnetwork.Subnet{
						Properties: &armnetwork.SubnetPropertiesFormat{
							NetworkSecurityGroup: &armnetwork.SecurityGroup{
								ID: &subnetNameWorkerNSGID, // in order to test calls to disable once per NSG
							},
						},
					},
				}, nil)
			},
			flowLogClientMock: func(client *mock_armnetwork.MockFlowLogsClientInterface) {
				client.EXPECT().DeleteAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameMasterNSGName, nil)
				client.EXPECT().DeleteAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameWorkerNSGName, nil)
			},
			instance: func(feature *aropreviewv1alpha1.PreviewFeature) {
				feature.Spec.NSGFlowLogs.Enabled = false
				feature.Spec.NSGFlowLogs.NetworkWatcherID = networkWatcherResourceId
			},
			wantErr: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			subnets := mock_armnetwork.NewMockSubnetsClient(controller)
			kubeSubnets := mock_subnet.NewMockKubeManager(controller)
			if tt.subnetMock != nil {
				tt.subnetMock(subnets, kubeSubnets)
			}

			instance := getValidPreviewFeatureInstance()
			if tt.instance != nil {
				tt.instance(instance)
			}

			flowLogsClient := mock_armnetwork.NewMockFlowLogsClientInterface(controller)
			if tt.flowLogClientMock != nil {
				tt.flowLogClientMock(flowLogsClient)
			}

			r := NewFeature(flowLogsClient, kubeSubnets, subnets, location)

			err := r.Reconcile(context.Background(), instance)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
