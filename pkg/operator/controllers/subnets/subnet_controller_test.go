package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_subnet "github.com/Azure/ARO-RP/pkg/util/mocks/subnet"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var (
	subscriptionId           = "0000000-0000-0000-0000-000000000000"
	clusterResourceGroupName = "aro-iljrzb5a"
	infraId                  = "abcd"
	clusterResourceGroupId   = "/subscriptions/" + subscriptionId + "/resourcegroups/" + clusterResourceGroupName
	vnetResourceGroup        = "vnet-rg"
	vnetName                 = "vnet"
	subnetNameWorker         = "worker"
	subnetNameMaster         = "master"
	subnetNameWorkerInvalid  = "worker-invalid"

	nsgv1NodeResourceId           = clusterResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + apisubnet.NSGNodeSuffixV1
	nsgv1MasterResourceId         = clusterResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + apisubnet.NSGControlPlaneSuffixV1
	nsgv2ResourceId               = clusterResourceGroupId + "/providers/Microsoft.Network/networkSecurityGroups/" + infraId + apisubnet.NSGSuffixV2
	subnetResourceIdMaster        = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
	subnetResourceIdWorker        = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
	subnetResourceIdWorkerInvalid = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker + "-invalid"
)

func getValidClusterInstance(operatorFlagEnabled bool, operatorFlagNSG bool, operatorFlagServiceEndpoint bool) *arov1alpha1.Cluster {
	return &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{
			ArchitectureVersion:    0,
			ClusterResourceGroupID: clusterResourceGroupId,
			InfraID:                infraId,
			OperatorFlags: arov1alpha1.OperatorFlags{
				operator.AzureSubnetsEnabled:     strconv.FormatBool(operatorFlagEnabled),
				operator.AzureSubnetsNsgManaged:  strconv.FormatBool(operatorFlagNSG),
				controllerServiceEndpointManaged: strconv.FormatBool(operatorFlagServiceEndpoint),
			},
		},
	}
}

func getValidSubnet() *armnetwork.Subnet {
	s := &armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			NetworkSecurityGroup: &armnetwork.SecurityGroup{
				ID: to.StringPtr(nsgv1MasterResourceId),
			},
			ServiceEndpoints: []*armnetwork.ServiceEndpointPropertiesFormat{},
		},
	}
	for _, endpoint := range api.SubnetsEndpoints {
		s.Properties.ServiceEndpoints = append(s.Properties.ServiceEndpoints, &armnetwork.ServiceEndpointPropertiesFormat{
			Service:           to.StringPtr(endpoint),
			ProvisioningState: ptr.To(armnetwork.ProvisioningStateSucceeded),
		})
	}
	return s
}

func TestReconcileManager(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name                        string
		subnetMock                  func(*mock_armnetwork.MockSubnetsClient, *mock_subnet.MockKubeManager)
		instance                    func(*arov1alpha1.Cluster)
		operatorFlagEnabled         bool
		operatorFlagNSG             bool
		operatorFlagServiceEndpoint bool
		wantAnnotationsUpdated      bool
		wantErr                     error
	}{
		{
			name:                        "Operator Disabled - no change",
			operatorFlagEnabled:         false,
			operatorFlagNSG:             false,
			operatorFlagServiceEndpoint: false,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetObjectWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)
			},
		},
		{
			name:                        "Architecture V1 - no change",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)
				//mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)
			},
		},
		{
			name:                        "Architecture V1 - all fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, *subnetObjectMasterUpdate, nil).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectWorkerUpdate, nil).Return(nil)
			},
		},
		{
			name:                        "Architecture V1 - skips invalid/not found subnets",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
					{
						ResourceID: subnetResourceIdWorkerInvalid,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, *subnetObjectMasterUpdate, nil).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectWorkerUpdate, nil).Return(nil)

				notFoundErr := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}

				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorkerInvalid, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorkerUpdate}, notFoundErr).AnyTimes()
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorkerInvalid, nil, gomock.Any()).Times(0)
			},
		},
		{
			name:                        "Architecture V1 - node only fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectWorkerUpdate, nil).Return(nil)
			},
		},
		{
			name:                        "Architecture V2 - no fixups",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)
			},
			instance: func(instance *arov1alpha1.Cluster) {
				instance.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - all nodes fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, *subnetObjectMasterUpdate, nil).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectWorkerUpdate, nil).Return(nil)
			},
			instance: func(instance *arov1alpha1.Cluster) {
				instance.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - skips invalid/not found subnets",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
					{
						ResourceID: subnetResourceIdWorkerInvalid,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, *subnetObjectMasterUpdate, nil).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectWorkerUpdate, nil).Return(nil)

				notFoundErr := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}

				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorkerInvalid, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorkerUpdate}, notFoundErr).AnyTimes()
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorkerInvalid, nil, gomock.Any()).Times(0)
			},
			instance: func(instance *arov1alpha1.Cluster) {
				instance.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - endpoint fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// master
				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.ServiceEndpoints = nil
				subnetObjectMaster.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)

				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				for i := range subnetObjectMasterUpdate.Properties.ServiceEndpoints {
					subnetObjectMasterUpdate.Properties.ServiceEndpoints[i].Locations = []*string{ptr.To("*")}
					subnetObjectMasterUpdate.Properties.ServiceEndpoints[i].ProvisioningState = nil
				}
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectMasterUpdate, nil).Return(nil)

				// worker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.ServiceEndpoints = nil
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()

				subnetObjectWorkerUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				for i := range subnetObjectWorkerUpdate.Properties.ServiceEndpoints {
					subnetObjectWorkerUpdate.Properties.ServiceEndpoints[i].Locations = []*string{ptr.To("*")}
					subnetObjectWorkerUpdate.Properties.ServiceEndpoints[i].ProvisioningState = nil
				}
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectWorkerUpdate, nil).Return(nil)
			},
			instance: func(instance *arov1alpha1.Cluster) {
				instance.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - no endpoint fixup because egress lockdown enabled",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				// master
				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.ServiceEndpoints = nil
				subnetObjectMaster.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				// worker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.ServiceEndpoints = nil
				subnetObjectWorker.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)
			},
			instance: func(instance *arov1alpha1.Cluster) {
				instance.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
				instance.Spec.GatewayDomains = []string{"somegatewaydomain.com"}
			},
		},
		{
			name:                        "Architecture V2 - empty NSG",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_armnetwork.MockSubnetsClient, kmock *mock_subnet.MockKubeManager) {
				kmock.EXPECT().List(gomock.Any()).Return([]subnet.Subnet{
					{
						ResourceID: subnetResourceIdMaster,
						IsMaster:   true,
					},
					{
						ResourceID: subnetResourceIdWorker,
						IsMaster:   false,
					},
				}, nil)

				subnetObjectMaster := getValidSubnet()
				subnetObjectMaster.Properties.NetworkSecurityGroup = nil
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectMaster}, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameMaster, *subnetObjectMasterUpdate, nil).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.Properties.NetworkSecurityGroup = nil
				mock.EXPECT().Get(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, nil).Return(armnetwork.SubnetsClientGetResponse{Subnet: *subnetObjectWorker}, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.Properties.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdateAndWait(gomock.Any(), vnetResourceGroup, vnetName, subnetNameWorker, *subnetObjectWorkerUpdate, nil).Return(nil)
			},
			instance: func(instance *arov1alpha1.Cluster) {
				instance.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
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

			instance := getValidClusterInstance(tt.operatorFlagEnabled, tt.operatorFlagNSG, tt.operatorFlagServiceEndpoint)
			if tt.instance != nil {
				tt.instance(instance)
			}

			clientFake := fake.NewClientBuilder().WithObjects(instance).Build()
			r := reconcileManager{
				log:            log,
				client:         clientFake,
				instance:       instance,
				subscriptionID: subscriptionId,
				subnets:        subnets,
				kubeSubnets:    kubeSubnets,
			}

			instanceCopy := *r.instance
			err := r.reconcileSubnets(context.Background())
			if err != nil {
				if tt.wantErr == nil {
					t.Fatal(err)
				}
				if err.Error() != tt.wantErr.Error() || err == nil && tt.wantErr != nil {
					t.Errorf("Expected Error %s, got %s when processing %s testcase", tt.wantErr.Error(), err.Error(), tt.name)
				}
			}

			if tt.wantAnnotationsUpdated && reflect.DeepEqual(instanceCopy, *r.instance) {
				t.Errorf("Expected annotations to be updated")
			}
		})
	}
}
