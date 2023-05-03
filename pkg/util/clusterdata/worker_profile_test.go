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
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	machinefake "github.com/openshift/client-go/machine/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"

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

var (
	clusterID = fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/group/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster",
		mockSubscriptionID,
	)
	workerSubnetID = fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		mockSubscriptionID, mockVnetRG, mockVnetName, mockSubnetName,
	)
)

// This function creates a new MachineSet object with the given name and ProviderSpec.
func createMachineSet(name string, ProvSpec machinev1beta1.ProviderSpec) *machinev1beta1.MachineSet {

	// Create a new MachineSet object with the given name and namespace.
	machset := &machinev1beta1.MachineSet{
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
	}

	// Return a pointer to the new MachineSet object.
	return machset
}

// This func returns a ProviderSpec object that represents a valid provider-specific configuration for a machine.
func validProvSpec() machinev1beta1.ProviderSpec {
	return machinev1beta1.ProviderSpec{
		Value: &kruntime.RawExtension{
			Raw: []byte(fmt.Sprintf(`{
    "apiVersion": "machine.openshift.io/v1beta1",
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
func getGotOc(clusterid string) *api.OpenShiftCluster {
	oc := &api.OpenShiftCluster{
		ID: clusterid,
	}
	return oc
}

// This function creates and returns an OpenShiftCluster object
// with the given worker profiles.
func getWantOc(workerprofile []api.WorkerProfile) *api.OpenShiftCluster {
	wantOc := &api.OpenShiftCluster{
		ID: clusterID,
		Properties: api.OpenShiftClusterProperties{
			WorkerProfiles: workerprofile,
		},
	}
	return wantOc
}

// This func returns an api.WorkerProfile object that represents a valid worker profile for a machine.
func validWorkerProfile() []api.WorkerProfile {
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

func TestWorkerProfilesEnricherTask2(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	invalidProvSpec := machinev1beta1.ProviderSpec{Value: &kruntime.RawExtension{
		Raw: []byte("invalid")}}

	emptyProvSpec := machinev1beta1.ProviderSpec{}
	noRawProvSpec := machinev1beta1.ProviderSpec{Value: &kruntime.RawExtension{}}

	invalidWorkerProfile := []api.WorkerProfile{{Name: "fake-worker-profile-1", Count: 1}}
	emptyWorkerProfile := []api.WorkerProfile{}

	testCases := []struct {
		name    string
		client  machineclient.Interface
		gotOc   *api.OpenShiftCluster
		wantOc  *api.OpenShiftCluster
		wantErr string
	}{
		{ //Test case: 1
			name:   "machine set objects exists - valid provider spec JSON",
			client: machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", validProvSpec()), createMachineSet("fake-worker-profile-2", validProvSpec())),
			wantOc: getWantOc(validWorkerProfile()),
			gotOc:  getGotOc(clusterID),
		},
		{ //Test case: 2
			name:   "machine set objects exists - invalid provider spec JSON",
			client: machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", invalidProvSpec)),
			wantOc: getWantOc(invalidWorkerProfile),
			gotOc:  getGotOc(clusterID),
		},
		{ //Test case: 3
			name:   "machine set objects exists - provider spec is missing",
			client: machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", emptyProvSpec)),
			wantOc: getWantOc(invalidWorkerProfile),
			gotOc:  getGotOc(clusterID),
		},
		{ //Test case: 4
			name:   "machine set objects exists - provider spec is missing raw value",
			client: machinefake.NewSimpleClientset(createMachineSet("fake-worker-profile-1", noRawProvSpec)),
			wantOc: getWantOc(invalidWorkerProfile),
			gotOc:  getGotOc(clusterID),
		},
		{ //Test case: 5
			name:   "machine set objects do not exist",
			client: machinefake.NewSimpleClientset(),
			wantOc: getWantOc(emptyWorkerProfile),
			gotOc:  getGotOc(clusterID),
		},
		{ //Test case: 6
			name:   "machine set list request failed",
			client: createFakeClientWithError(),
			wantOc: &api.OpenShiftCluster{
				ID: clusterID,
			},
			gotOc:   getGotOc(clusterID),
			wantErr: "fake list error",
		},
		{ //Test case: 7
			name:   "invalid cluster object",
			client: machinefake.NewSimpleClientset(),
			wantOc: &api.OpenShiftCluster{
				ID: "invalid",
			},
			gotOc:   getGotOc("invalid"),
			wantErr: "parsing failed for invalid. Invalid resource Id format",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//Given
			// Create a clients object with the given client object
			clients := clients{
				machine: tc.client,
			}

			// Create a machineClientEnricher instance and call the SetDefaults method with the given gotOc object
			e := machineClientEnricher{}
			e.SetDefaults(tc.gotOc)

			//When
			// Call the Enrich method on the machineClientEnricher instance with the given clients objects
			err := e.Enrich(context.Background(), log, tc.gotOc, clients.k8s, clients.config, clients.machine, clients.operator)

			//Then
			// Check if the returned error matches the expected error using the AssertErrorMessage function
			errorHandling.AssertErrorMessage(t, err, tc.wantErr)
			// Check if the gotOc object after enrichment is equal to the expected wantOc object using the DeepEqual function
			if !reflect.DeepEqual(tc.gotOc, tc.wantOc) {
				// If they are not equal, report the differences between the two objects using the cmp.Diff function
				t.Error(cmp.Diff(tc.gotOc, tc.wantOc))
			}
		})
	}

}
