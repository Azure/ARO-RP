package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	errorHandling "github.com/Azure/ARO-RP/test/util/error"
)

const (
	mockSubscriptionID = "00000000-0000-0000-0000-000000000000"
	mockVnetRG         = "fake-vnet-rg"
	mockVnetName       = "fake-vnet"
	mockSubnetName     = "cluster-worker"
)

func TestWorkerProfilesEnricherTask(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	var clusterID = fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/group/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster",
		mockSubscriptionID,
	)

	invalidProvSpec := machinev1beta1.ProviderSpec{Value: &kruntime.RawExtension{
		Raw: []byte("invalid")}}

	emptyProvSpec := machinev1beta1.ProviderSpec{}
	noRawProvSpec := machinev1beta1.ProviderSpec{Value: &kruntime.RawExtension{}}

	invalidWorkerProfile := []api.WorkerProfile{{Name: "fake-worker-profile-1", Count: 1}}
	emptyWorkerProfile := []api.WorkerProfile{}

	testCases := []struct {
		name    string
		client  machineclient.Interface
		givenOc *api.OpenShiftCluster
		wantOc  *api.OpenShiftCluster
		wantErr string
	}{
		{
			name:    "machine set objects exist - valid provider spec JSON",
			client:  machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", validProvSpec()), createMachineSet("fake-worker-profile-2", validProvSpec())),
			wantOc:  getWantOc(clusterID, validWorkerProfile()),
			givenOc: getGivenOc(clusterID),
		},
		{
			name:    "machine set objects exist - invalid provider spec JSON - zone as int - treated as valid",
			client:  machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", validProvSpec()), createMachineSet("fake-worker-profile-2", invalidProvSpecZoneAsInt())),
			wantOc:  getWantOc(clusterID, validWorkerProfile()),
			givenOc: getGivenOc(clusterID),
		},
		{
			name:    "machine set objects exist - invalid provider spec JSON - tag as int - treated as valid",
			client:  machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", validProvSpec()), createMachineSet("fake-worker-profile-2", invalidProvSpecTagsAsInt())),
			wantOc:  getWantOc(clusterID, validWorkerProfile()),
			givenOc: getGivenOc(clusterID),
		},
		{
			name:    "machine set objects exist - invalid provider spec JSON",
			client:  machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", invalidProvSpec)),
			wantOc:  getWantOc(clusterID, invalidWorkerProfile),
			givenOc: getGivenOc(clusterID),
		},
		{
			name:    "machine set objects exist - provider spec is missing",
			client:  machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", emptyProvSpec)),
			wantOc:  getWantOc(clusterID, invalidWorkerProfile),
			givenOc: getGivenOc(clusterID),
		},
		{
			name:    "machine set objects exist - provider spec is missing raw value",
			client:  machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", noRawProvSpec)),
			wantOc:  getWantOc(clusterID, invalidWorkerProfile),
			givenOc: getGivenOc(clusterID),
		},
		{
			name: "machine set objects exist - machineset has no ready replicas",
			client: machinefake.NewSimpleClientset(func() *machinev1beta1.MachineSet {
				ms := createMachineSet("fake-worker-profile-1", validProvSpec())
				ms.Status.ReadyReplicas = 0
				return ms
			}()),
			wantOc:  getWantOc(clusterID, invalidWorkerProfile),
			givenOc: getGivenOc(clusterID),
		},
		{
			name:    "machine set objects do not exist",
			client:  machinefake.NewSimpleClientset(),
			wantOc:  getWantOc(clusterID, emptyWorkerProfile),
			givenOc: getGivenOc(clusterID),
		},
		{
			name:   "machine set list request failed",
			client: createFakeClientWithError(),
			wantOc: &api.OpenShiftCluster{
				ID: clusterID,
			},
			givenOc: getGivenOc(clusterID),
			wantErr: "fake list error",
		},
		{
			name:   "invalid cluster object",
			client: machinefake.NewSimpleClientset(),
			wantOc: &api.OpenShiftCluster{
				ID: "invalid",
			},
			givenOc: getGivenOc("invalid"),
			wantErr: "parsing failed for invalid. Invalid resource Id format",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//Given
			clients := clients{
				machine: tc.client,
			}

			e := machineClientEnricher{}

			//When
			e.SetDefaults(tc.givenOc)
			err := e.Enrich(context.Background(), log, tc.givenOc, clients.k8s, clients.config, clients.machine, clients.operator)

			//Then
			errorHandling.AssertErrorMessage(t, err, tc.wantErr)

			if !reflect.DeepEqual(tc.givenOc, tc.wantOc) {
				t.Error(cmp.Diff(tc.givenOc, tc.wantOc, gocmp.AllowUnexported(sync.Mutex{})))
			}
		})
	}
}

// This function creates a new MachineSet object with the given name and ProviderSpec.
func createMachineSet(name string, ProvSpec machinev1beta1.ProviderSpec) *machinev1beta1.MachineSet {
	return &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "openshift-machine-api",
		},

		// Specify the desired state for the MachineSet
		Spec: machinev1beta1.MachineSetSpec{
			Replicas: to.Int32Ptr(1),
			Template: machinev1beta1.MachineTemplateSpec{
				// Specify the desired configuration for the machine using ProviderSpec
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: ProvSpec,
				},
			},
		},
		Status: machinev1beta1.MachineSetStatus{
			ReadyReplicas: 1,
		},
	}
}

// This func returns a ProviderSpec object that represents a valid provider-specific configuration for a machine.
func validProvSpec() machinev1beta1.ProviderSpec {
	return machinev1beta1.ProviderSpec{
		Value: &kruntime.RawExtension{
			Raw: []byte(fmt.Sprintf(`{
	"apiVersion": "machine.openshift.io/v1beta1",
	"kind": "AzureMachineProviderSpec",
	"tags": {
		"field1": "value1",
		"field2": "value2"
	},
	"osDisk": {
		"diskSizeGB": 512
	},
	"vmSize": "Standard_D4s_v3",
	"networkResourceGroup": "%s",
	"vnet": "%s",
	"subnet": "%s",
	"zone": "1"
}`,
				mockVnetRG, mockVnetName, mockSubnetName,
			)),
		},
	}
}

func invalidProvSpecZoneAsInt() machinev1beta1.ProviderSpec {
	return machinev1beta1.ProviderSpec{
		Value: &kruntime.RawExtension{
			Raw: []byte(fmt.Sprintf(`{
	"apiVersion": "machine.openshift.io/v1beta1",
	"kind": "AzureMachineProviderSpec",
	"tags": {
		"field1": "value1",
		"field2": "value2"
	},
	"osDisk": {
		"diskSizeGB": 512
	},
	"vmSize": "Standard_D4s_v3",
	"networkResourceGroup": "%s",
	"vnet": "%s",
	"subnet": "%s",
	"zone": 1
}`,
				mockVnetRG, mockVnetName, mockSubnetName,
			)),
		},
	}
}

func invalidProvSpecTagsAsInt() machinev1beta1.ProviderSpec {
	return machinev1beta1.ProviderSpec{
		Value: &kruntime.RawExtension{
			Raw: []byte(fmt.Sprintf(`{
	"apiVersion": "machine.openshift.io/v1beta1",
	"kind": "AzureMachineProviderSpec",
	"tags": {
		"field1": "value1",
		"field2": 2
	},
	"osDisk": {
		"diskSizeGB": 512
	},
	"vmSize": "Standard_D4s_v3",
	"networkResourceGroup": "%s",
	"vnet": "%s",
	"subnet": "%s",
	"zone": "1"
}`,
				mockVnetRG, mockVnetName, mockSubnetName,
			)),
		},
	}
}

// This function creates a fake client with a reactor function
// that returns an error when listing MachineSets.
func createFakeClientWithError() machineclient.Interface {
	client := machinefake.NewSimpleClientset()
	client.PrependReactor("list", "machinesets", func(action ktesting.Action) (bool, kruntime.Object, error) {
		return true, nil, errors.New("fake list error")
	})
	return client
}

// This func creates and returns an OpenShiftCluster object with the given clusterid.
func getGivenOc(clusterid string) *api.OpenShiftCluster {
	return &api.OpenShiftCluster{
		ID: clusterid,
	}
}

// This function creates and returns an OpenShiftCluster object
// with the given worker profiles.
func getWantOc(clusID string, workerprofile []api.WorkerProfile) *api.OpenShiftCluster {
	return &api.OpenShiftCluster{
		ID: clusID,
		Properties: api.OpenShiftClusterProperties{
			WorkerProfilesStatus: workerprofile,
		},
	}
}

// This func returns an api.WorkerProfile object that represents a valid worker profile for a machine.
func validWorkerProfile() []api.WorkerProfile {
	var workerSubnetID = fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		mockSubscriptionID, mockVnetRG, mockVnetName, mockSubnetName,
	)

	return []api.WorkerProfile{
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
			VMSize:           api.VMSizeStandardD4sV3,
			DiskSizeGB:       512,
			EncryptionAtHost: api.EncryptionAtHostDisabled,
			SubnetID:         workerSubnetID,
			Count:            1,
		},
	}
}
