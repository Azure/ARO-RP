package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	machinev1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	clusterapi "github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset"
	"github.com/openshift/cluster-api/pkg/client/clientset_generated/clientset/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/test/util/cmp"
)

func TestWorkerProfilesEnricherTask(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	mockSubscriptionID := "00000000-0000-0000-0000-000000000000"
	mockVnetRG := "fake-vnet-rg"
	mockVnetName := "fake-vnet"
	mockSubnetName := "cluster-worker"
	clusterID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/group/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster",
		mockSubscriptionID,
	)
	workerSubnetID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		mockSubscriptionID, mockVnetRG, mockVnetName, mockSubnetName,
	)

	for _, tt := range []struct {
		name     string
		client   func() clusterapi.Interface
		modifyOc func(*api.OpenShiftCluster)
		wantOc   *api.OpenShiftCluster
		wantErr  string
	}{
		{
			name: "machine set objects exists - valid provider spec JSON",
			client: func() clusterapi.Interface {
				return fake.NewSimpleClientset(
					&machinev1beta1.MachineSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-worker-profile-1",
							Namespace: "openshift-machine-api",
						},
						Spec: machinev1beta1.MachineSetSpec{
							Template: machinev1beta1.MachineTemplateSpec{
								Spec: machinev1beta1.MachineSpec{
									ProviderSpec: machinev1beta1.ProviderSpec{
										Value: &runtime.RawExtension{
											Raw: []byte(fmt.Sprintf(`{
	"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
	"kind": "AzureMachineProviderSpec",
	"osDisk": {
		"diskSizeGB": 512
	},
	"vmSize": "Standard_D4s_v3",
	"networkResourceGroup": "%s",
	"vnet": "%s",
	"subnet": "%s"
}`,
												mockVnetRG, mockVnetName, mockSubnetName,
											)),
										},
									},
								},
							},
						},
					},
					&machinev1beta1.MachineSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-worker-profile-2",
							Namespace: "openshift-machine-api",
						},
						Spec: machinev1beta1.MachineSetSpec{
							Replicas: int32Ptr(2),
							Template: machinev1beta1.MachineTemplateSpec{
								Spec: machinev1beta1.MachineSpec{
									ProviderSpec: machinev1beta1.ProviderSpec{
										Value: &runtime.RawExtension{
											Raw: []byte(fmt.Sprintf(`{
	"apiVersion": "azureproviderconfig.openshift.io/v1beta1",
	"kind": "AzureMachineProviderSpec",
	"osDisk": {
		"diskSizeGB": 128
	},
	"vmSize": "Standard_D2s_v3",
	"networkResourceGroup": "%s",
	"vnet": "%s",
	"subnet": "%s"
}`,
												mockVnetRG, mockVnetName, mockSubnetName,
											)),
										},
									},
								},
							},
						},
					},
				)
			},
			wantOc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					WorkerProfiles: []api.WorkerProfile{
						{
							Name:       "fake-worker-profile-1",
							VMSize:     api.VMSizeStandardD4sV3,
							DiskSizeGB: 512,
							SubnetID:   workerSubnetID,
							Count:      1,
						},
						{
							Name:       "fake-worker-profile-2",
							VMSize:     api.VMSizeStandardD2sV3,
							DiskSizeGB: 128,
							SubnetID:   workerSubnetID,
							Count:      2,
						},
					},
				},
			},
		}, {
			name: "machine set objects exists - invalid provider spec JSON",
			client: func() clusterapi.Interface {
				return fake.NewSimpleClientset(
					&machinev1beta1.MachineSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-worker-profile-1",
							Namespace: "openshift-machine-api",
						},
						Spec: machinev1beta1.MachineSetSpec{
							Template: machinev1beta1.MachineTemplateSpec{
								Spec: machinev1beta1.MachineSpec{
									ProviderSpec: machinev1beta1.ProviderSpec{
										Value: &runtime.RawExtension{
											Raw: []byte("invalid"),
										},
									},
								},
							},
						},
					},
				)
			},
			wantOc: &api.OpenShiftCluster{
				ID: clusterID,
			},
			wantErr: `couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string "json:\"apiVersion,omitempty\""; Kind string "json:\"kind,omitempty\"" }`,
		},
		{
			name: "machine set objects do not exist",
			client: func() clusterapi.Interface {
				return fake.NewSimpleClientset()
			},
			wantOc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					WorkerProfiles: []api.WorkerProfile{},
				},
			},
		},
		{
			name: "machine set list request failed",
			client: func() clusterapi.Interface {
				client := fake.NewSimpleClientset()
				client.PrependReactor("list", "machinesets", func(action clientgotesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("fake list error")
				})
				return client
			},
			wantOc: &api.OpenShiftCluster{
				ID: clusterID,
			},
			wantErr: "fake list error",
		},
		{
			name: "invalid cluster object",
			client: func() clusterapi.Interface {
				return fake.NewSimpleClientset()
			},
			modifyOc: func(oc *api.OpenShiftCluster) {
				oc.ID = "invalid"
			},
			wantOc: &api.OpenShiftCluster{
				ID: "invalid",
			},
			wantErr: "parsing failed for invalid. Invalid resource Id format",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oc := &api.OpenShiftCluster{
				ID: clusterID,
			}

			if tt.modifyOc != nil {
				tt.modifyOc(oc)
			}

			e := &workerProfilesEnricherTask{
				log:    log,
				client: tt.client(),
				oc:     oc,
			}
			e.SetDefaults()

			callbacks := make(chan func())
			errors := make(chan error)
			go e.FetchData(callbacks, errors)

			select {
			case f := <-callbacks:
				f()
				if !reflect.DeepEqual(oc, tt.wantOc) {
					t.Error(cmp.Diff(oc, tt.wantOc))
				}
			case err := <-errors:
				if tt.wantErr != err.Error() {
					t.Error(err)
				}
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
