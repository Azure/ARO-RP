package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
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

func getValidSubnet() *mgmtnetwork.Subnet {
	s := &mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
			NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
				ID: to.StringPtr(nsgv1MasterResourceId),
			},
			ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{},
		},
	}
	for _, endpoint := range api.SubnetsEndpoints {
		*s.ServiceEndpoints = append(*s.ServiceEndpoints, mgmtnetwork.ServiceEndpointPropertiesFormat{
			Service:           to.StringPtr(endpoint),
			ProvisioningState: mgmtnetwork.Succeeded,
		})
	}
	return s
}

func TestReconcileManager(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name                        string
		subnetMock                  func(*mock_subnet.MockManager, *mock_subnet.MockKubeManager)
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
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)
			},
		},
		{
			name:                        "Architecture V1 - no change",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)
			},
		},
		{
			name:                        "Architecture V1 - all fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdMaster, subnetObjectMasterUpdate).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
		},
		{
			name:                        "Architecture V1 - skips invalid/not found subnets",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1MasterResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdMaster, subnetObjectMasterUpdate).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectWorkerUpdate).Return(nil)

				notFoundErr := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}

				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorkerInvalid).Return(nil, notFoundErr).AnyTimes()
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorkerInvalid, gomock.Any()).Times(0)
			},
		},
		{
			name:                        "Architecture V1 - node only fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv1NodeResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
		},
		{
			name:                        "Architecture V2 - no fixups",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - all nodes fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdMaster, subnetObjectMasterUpdate).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - skips invalid/not found subnets",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdMaster, subnetObjectMasterUpdate).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId + "new")
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectWorkerUpdate).Return(nil)

				notFoundErr := autorest.DetailedError{
					StatusCode: http.StatusNotFound,
				}

				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorkerInvalid).Return(nil, notFoundErr).AnyTimes()
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorkerInvalid, gomock.Any()).Times(0)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - endpoint fixup",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.ServiceEndpoints = nil
				subnetObjectMaster.ServiceEndpoints = nil
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				for i := range *subnetObjectMasterUpdate.ServiceEndpoints {
					(*subnetObjectMasterUpdate.SubnetPropertiesFormat.ServiceEndpoints)[i].Locations = &[]string{"*"}
					(*subnetObjectMasterUpdate.SubnetPropertiesFormat.ServiceEndpoints)[i].ProvisioningState = ""
				}
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectMasterUpdate).Return(nil)

				// worker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.ServiceEndpoints = nil
				subnetObjectWorker.ServiceEndpoints = nil
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				for i := range *subnetObjectWorkerUpdate.ServiceEndpoints {
					(*subnetObjectWorkerUpdate.SubnetPropertiesFormat.ServiceEndpoints)[i].Locations = &[]string{"*"}
					(*subnetObjectWorkerUpdate.SubnetPropertiesFormat.ServiceEndpoints)[i].ProvisioningState = ""
				}
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
		},
		{
			name:                        "Architecture V2 - no endpoint fixup because egress lockdown enabled",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      false,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.ServiceEndpoints = nil
				subnetObjectMaster.ServiceEndpoints = nil
				subnetObjectMaster.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectMaster, nil).MaxTimes(2)

				// worker
				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.ServiceEndpoints = nil
				subnetObjectWorker.ServiceEndpoints = nil
				subnetObjectWorker.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
				instace.Spec.GatewayDomains = []string{"somegatewaydomain.com"}
			},
		},
		{
			name:                        "Architecture V2 - empty NSG",
			operatorFlagEnabled:         true,
			operatorFlagNSG:             true,
			operatorFlagServiceEndpoint: true,
			wantAnnotationsUpdated:      true,
			subnetMock: func(mock *mock_subnet.MockManager, kmock *mock_subnet.MockKubeManager) {
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
				subnetObjectMaster.NetworkSecurityGroup = nil
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdMaster).Return(subnetObjectMaster, nil).MaxTimes(2)

				subnetObjectMasterUpdate := getValidSubnet()
				subnetObjectMasterUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdMaster, subnetObjectMasterUpdate).Return(nil)

				subnetObjectWorker := getValidSubnet()
				subnetObjectWorker.NetworkSecurityGroup = nil
				mock.EXPECT().Get(gomock.Any(), subnetResourceIdWorker).Return(subnetObjectWorker, nil).MaxTimes(2)

				subnetObjectWorkerUpdate := getValidSubnet()
				subnetObjectWorkerUpdate.NetworkSecurityGroup.ID = to.StringPtr(nsgv2ResourceId)
				mock.EXPECT().CreateOrUpdate(gomock.Any(), subnetResourceIdWorker, subnetObjectWorkerUpdate).Return(nil)
			},
			instance: func(instace *arov1alpha1.Cluster) {
				instace.Spec.ArchitectureVersion = int(api.ArchitectureVersionV2)
			},
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
