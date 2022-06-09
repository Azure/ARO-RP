package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
)

var machineAzureProviderSpec = &machinev1beta1.AzureMachineProviderSpec{
	Image: machinev1beta1.Image{
		Publisher: "azureopenshift",
		Offer:     "aro4",
		SKU:       "aro_499",
		Version:   "499.99.30000610",
	},
	Location:             "neversayneverland",
	NetworkResourceGroup: "networkRG",
	OSDisk: machinev1beta1.OSDisk{
		DiskSizeGB: int32(9001),
		OSType:     "Linux",
		ManagedDisk: machinev1beta1.ManagedDiskParameters{
			StorageAccountType: "Premium_LRS",
		},
	},
	PublicIP:           false,
	PublicLoadBalancer: "",
	ResourceGroup:      "myRG",
	VMSize:             "Standard_D8s_v3",
	Vnet:               "myVnet",
	Zone:               to.StringPtr("2"),
}

var azAzureProviderSpec = &azureproviderv1beta1.AzureMachineProviderSpec{
	Image: azureproviderv1beta1.Image{
		Publisher: "azureopenshift",
		Offer:     "aro4",
		SKU:       "aro_499",
		Version:   "499.99.30000610",
	},
	Location:             "neversayneverland",
	NetworkResourceGroup: "networkRG",
	OSDisk: azureproviderv1beta1.OSDisk{
		DiskSizeGB: int32(9001),
		OSType:     "Linux",
		ManagedDisk: azureproviderv1beta1.ManagedDiskParameters{
			StorageAccountType: "Premium_LRS",
		},
	},
	PublicIP:           false,
	PublicLoadBalancer: "",
	ResourceGroup:      "myRG",
	VMSize:             "Standard_D8s_v3",
	Vnet:               "myVnet",
	Zone:               to.StringPtr("2"),
}

func marshalAzureMachineProviderSpec(t *testing.T, spec kruntime.Object) []byte {
	serializer := kjson.NewSerializerWithOptions(
		kjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme,
		kjson.SerializerOptions{Yaml: false},
	)

	json := scheme.Codecs.CodecForVersions(serializer, nil, schema.GroupVersions(scheme.Scheme.PrioritizedVersionsAllGroups()), nil)

	buf := &bytes.Buffer{}
	err := json.Encode(spec, buf)
	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestUnmarshalAzureProviderSpec(t *testing.T) {
	machineName := "myMachine"

	// Register old type so we can encode the CR to bytes for use only in test
	scheme.Scheme.AddKnownTypes(azureproviderv1beta1.SchemeGroupVersion, &azureproviderv1beta1.AzureMachineProviderSpec{})

	machineProviderSpecBytes := marshalAzureMachineProviderSpec(t, machineAzureProviderSpec)
	azProviderSpecBytes := marshalAzureMachineProviderSpec(t, azAzureProviderSpec)

	for _, tt := range []struct {
		name         string
		machineType  MachineType
		providerSpec []byte
		wantErr      string
	}{
		{
			name:         "valid - machine.openshift.io provider spec",
			providerSpec: machineProviderSpecBytes,
		},
		{
			name:         "valid - azureproviderconfig.openshift.io provider spec",
			providerSpec: azProviderSpecBytes,
		},
		{
			name:         "fail - azureproviderconfig.openshift.io invalid json",
			providerSpec: []byte("azureproviderconfig.openshift.io"),
			machineType:  MachineSet,
			wantErr:      fmt.Sprintf("%s %s: failed to unmarshal the 'azureproviderconfig.openshift.io' provider spec: %q", MachineSet, machineName, "invalid character 'a' looking for beginning of value"),
		},
		{
			name:         "fail - unable to decode type",
			providerSpec: []byte(`{"apiVersion": "invalid/v1beta1","kind": "AzureMachineProviderSpec"}`),
			machineType:  MachineSet,
			wantErr:      `no kind "AzureMachineProviderSpec" is registered for version "invalid/v1beta1" in scheme`,
		},
		{
			name:         "fail - invalid type",
			providerSpec: []byte(`{"apiVersion": "v1","kind": "Node"}`),
			machineType:  MachineSet,
			wantErr:      `failed to read provider spec:`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			actualProviderSpec, err := UnmarshalAzureProviderSpec(machineName, tt.machineType, tt.providerSpec)
			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}

				// Ignoring check to support both azureproviderconfig and machine providerSpec APIVersions
				actualProviderSpec.TypeMeta = metav1.TypeMeta{}

				if !reflect.DeepEqual(actualProviderSpec, machineAzureProviderSpec) {
					t.Errorf("got '%v' wanted '%v'", spew.Sdump(actualProviderSpec), spew.Sdump(machineAzureProviderSpec))
				}
			} else {
				// Use contains as one test case contains line numbers which may change
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Error(err)
				}
			}
		})
	}
}
