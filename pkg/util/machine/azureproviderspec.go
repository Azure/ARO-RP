package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"strings"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"

	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type MachineType string

const (
	MachineSet        MachineType = "machineset"
	Machine           MachineType = "machine"
	azureProviderSpec             = "azureproviderconfig.openshift.io"
)

// UnmarshalAzureProviderSpec unmarshals an Azure Provider Spec set on machines or machinesets
// it contains backward compatibility for the older azureproviderconfig which is no longer in use
// in OCP 4.10+
func UnmarshalAzureProviderSpec(name string, mType MachineType, rawProviderSpec []byte) (*machinev1beta1.AzureMachineProviderSpec, error) {
	if strings.Contains(string(rawProviderSpec), azureProviderSpec) {
		machineProviderSpec := &machinev1beta1.AzureMachineProviderSpec{}
		err := json.Unmarshal(rawProviderSpec, machineProviderSpec)
		if err != nil {
			return nil, fmt.Errorf("%s %s: failed to unmarshal the %q provider spec: %q", mType, name, azureProviderSpec, err.Error())
		}
		return machineProviderSpec, nil
	}

	o, _, err := scheme.Codecs.UniversalDeserializer().Decode(rawProviderSpec, nil, nil)
	if err != nil {
		return nil, err
	}

	machineProviderSpec, ok := o.(*machinev1beta1.AzureMachineProviderSpec)
	if !ok {
		// If this happens, the azure machine provider spec type/apiversion may have been updated and
		// we need to handle it appropriately
		return nil, fmt.Errorf("%s %s: failed to read provider spec: %T", mType, name, o)
	}

	return machineProviderSpec, nil
}
