package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	maofake "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
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
		client   func() maoclient.Interface
		modifyOc func(*api.OpenShiftCluster)
		wantOc   *api.OpenShiftCluster
		wantErr  string
	}{
		{
			name: "machine set objects exists - valid provider spec JSON",
			client: func() maoclient.Interface {
				return maofake.NewSimpleClientset(
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
							Replicas: to.Int32Ptr(2),
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
							Name:             "fake-worker-profile-1",
							VMSize:           api.VMSizeStandardD4sV3,
							DiskSizeGB:       512,
							EncryptionAtHost: api.EncryptionAtHostDisabled,
							SubnetID:         workerSubnetID,
							Count:            1,
						},
						{
							Name:             "fake-worker-profile-2",
							VMSize:           api.VMSizeStandardD2sV3,
							DiskSizeGB:       128,
							EncryptionAtHost: api.EncryptionAtHostDisabled,
							SubnetID:         workerSubnetID,
							Count:            2,
						},
					},
				},
			},
		},
		{
			name: "machine set objects exists - invalid provider spec JSON",
			client: func() maoclient.Interface {
				return maofake.NewSimpleClientset(
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
				Properties: api.OpenShiftClusterProperties{
					WorkerProfiles: []api.WorkerProfile{{Name: "fake-worker-profile-1", Count: 1}},
				},
			},
		},
		{
			name: "machine set objects exists - provider spec is missing",
			client: func() maoclient.Interface {
				return maofake.NewSimpleClientset(
					&machinev1beta1.MachineSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-worker-profile-1",
							Namespace: "openshift-machine-api",
						},
					},
				)
			},
			wantOc: &api.OpenShiftCluster{
				ID: clusterID,
				Properties: api.OpenShiftClusterProperties{
					WorkerProfiles: []api.WorkerProfile{{Name: "fake-worker-profile-1", Count: 1}},
				},
			},
		},
		{
			name: "machine set objects exists - provider spec is missing raw value",
			client: func() maoclient.Interface {
				return maofake.NewSimpleClientset(
					&machinev1beta1.MachineSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-worker-profile-1",
							Namespace: "openshift-machine-api",
						},
						Spec: machinev1beta1.MachineSetSpec{
							Template: machinev1beta1.MachineTemplateSpec{
								Spec: machinev1beta1.MachineSpec{
									ProviderSpec: machinev1beta1.ProviderSpec{
										Value: &runtime.RawExtension{},
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
					WorkerProfiles: []api.WorkerProfile{{Name: "fake-worker-profile-1", Count: 1}},
				},
			},
		},
		{
			name: "machine set objects do not exist",
			client: func() maoclient.Interface {
				return maofake.NewSimpleClientset()
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
			client: func() maoclient.Interface {
				client := maofake.NewSimpleClientset()
				client.PrependReactor("list", "machinesets", func(action ktesting.Action) (bool, runtime.Object, error) {
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
			client: func() maoclient.Interface {
				return maofake.NewSimpleClientset()
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
				maocli: tt.client(),
				oc:     oc,
			}
			e.SetDefaults()

			callbacks := make(chan func())
			errors := make(chan error)
			go e.FetchData(context.Background(), callbacks, errors)

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
