package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
)

func masterSubnetId(ctx context.Context, clustercli maoclient.Interface, vnetID string) (*string, error) {
	machines, err := clustercli.MachineV1beta1().Machines(machineSetsNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var masterSubnet string

	for _, machine := range machines.Items {
		if isMaster, err := isMasterRole(&machine); err != nil && isMaster {
			o, _, err := scheme.Codecs.UniversalDeserializer().Decode(machine.Spec.ProviderSpec.Value.Raw, nil, nil)
			if err != nil {
				return nil, err
			}

			machineProviderSpec, ok := o.(*azureproviderv1beta1.AzureMachineProviderSpec)
			if !ok {
				return nil, fmt.Errorf("machine %s: failed to read provider spec: %T", machine.Name, o)
			}
			masterSubnet = machineProviderSpec.Subnet
			break
		}
	}
	result := fmt.Sprintf("%s/subnets/%s", vnetID, masterSubnet)

	return &result, nil
}
