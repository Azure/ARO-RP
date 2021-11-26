package nsgflowlogs

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"
	mock_network "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/network"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
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

func getValidFlowLogFeature() *mgmtnetwork.FlowLog {
	return &mgmtnetwork.FlowLog{
		FlowLogPropertiesFormat: &mgmtnetwork.FlowLogPropertiesFormat{
			TargetResourceID: to.StringPtr(""),
			Enabled:          to.BoolPtr(true),
			StorageID:        to.StringPtr(""),
			RetentionPolicy: &mgmtnetwork.RetentionPolicyParameters{
				Days: to.Int32Ptr(0),
			},
			Format: &mgmtnetwork.FlowLogFormatParameters{
				Type:    mgmtnetwork.JSON,
				Version: to.Int32Ptr(0),
			},
			FlowAnalyticsConfiguration: &mgmtnetwork.TrafficAnalyticsProperties{
				NetworkWatcherFlowAnalyticsConfiguration: &mgmtnetwork.TrafficAnalyticsConfigurationProperties{
					TrafficAnalyticsInterval: to.Int32Ptr(0),
					WorkspaceID:              to.StringPtr(""),
				},
			},
			ProvisioningState: "",
		},
		Location: to.StringPtr(location),
	}
}

func TestReconcileManager(t *testing.T) {
	for _, tt := range []struct {
		name              string
		subnetMock        func(*mock_subnet.MockManager, *mock_subnet.MockKubeManager)
		instance          func(*aropreviewv1alpha1.PreviewFeature)
		flowLogClientMock func(*mock_network.MockFlowLogsClient)
		wantErr           string
	}{
		{
			name: "do not enable flow log if parameters are missing/wrong",
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: resourceIdMaster,
					},
					{
						ResourceID: resourceIdWorker,
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameMasterNSGID,
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameWorkerNSGID,
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
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameMasterNSGID,
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameMasterNSGID, // same NSG as the master subnet
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker2).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameWorkerNSGID, // different NSG ID. expect another one call to create
						},
					},
				}, nil)
			},
			flowLogClientMock: func(client *mock_network.MockFlowLogsClient) {
				flowLogMaster := getValidFlowLogFeature()
				flowLogMaster.FlowLogPropertiesFormat.TargetResourceID = &subnetNameMasterNSGID

				flowLogWorker := getValidFlowLogFeature()
				flowLogWorker.FlowLogPropertiesFormat.TargetResourceID = &subnetNameWorkerNSGID
				// enable once per NSG
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameMasterNSGName, *flowLogMaster)
				client.EXPECT().CreateOrUpdateAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameWorkerNSGName, *flowLogWorker)
			},
			instance: func(feature *aropreviewv1alpha1.PreviewFeature) {
				feature.Spec.NSGFlowLogs.Enabled = true
				feature.Spec.NSGFlowLogs.NetworkWatcherID = networkWatcherResourceId
			},
			wantErr: "",
		},
		{
			name: "disable flow log",
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				mock.EXPECT().Get(gomock.Any(), resourceIdMaster).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameMasterNSGID,
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameMasterNSGID, // same NSG as the master subnet
						},
					},
				}, nil)
				mock.EXPECT().Get(gomock.Any(), resourceIdWorker2).Return(&mgmtnetwork.Subnet{
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
							ID: &subnetNameWorkerNSGID, // in order to test calls to disable once per NSG
						},
					},
				}, nil)
			},
			flowLogClientMock: func(client *mock_network.MockFlowLogsClient) {
				client.EXPECT().DeleteAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameMasterNSGName)
				client.EXPECT().DeleteAndWait(gomock.Any(), networkWatcherResourceGroupName, networkWatcherName, subnetNameWorkerNSGName)
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

			subnets := mock_subnet.NewMockManager(controller)
			kubeSubnets := mock_subnet.NewMockKubeManager(controller)
			if tt.subnetMock != nil {
				tt.subnetMock(subnets, kubeSubnets)
			}

			instance := getValidPreviewFeatureInstance()
			if tt.instance != nil {
				tt.instance(instance)
			}

			flowLogsClient := mock_network.NewMockFlowLogsClient(controller)
			if tt.flowLogClientMock != nil {
				tt.flowLogClientMock(flowLogsClient)
			}

			r := NewFeature(flowLogsClient, kubeSubnets, subnets, location)

			err := r.Reconcile(context.Background(), instance)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
