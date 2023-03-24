package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubeManager interface interact with kubernetes layer to extract required information
type KubeManager interface {
	List(ctx context.Context) ([]Subnet, error)
}

type kubeManager struct {
	client client.Client

	subscriptionID string
}

func NewKubeManager(client client.Client, subscriptionID string) KubeManager {
	return &kubeManager{
		client:         client,
		subscriptionID: subscriptionID,
	}
}

// List reconstructs subnetId used in machines object in the cluster
// In cases when we interact with customer vnets, we don't know which subnets are used in ARO.
// Example : /subscriptions/{subscriptionID}/resourceGroups/{vnet-resource-group}/providers/Microsoft.Network/virtualNetworks/{vnet-name}/subnets/{subnet-name}
func (m *kubeManager) List(ctx context.Context) ([]Subnet, error) {
	subnetMap := []Subnet{}

	// select all workers by the  machine.openshift.io/cluster-api-machine-role: not equal to master Label
	selector, _ := labels.Parse("machine.openshift.io/cluster-api-machine-role!=master")
	machines := &machinev1beta1.MachineList{}
	err := m.client.List(ctx, machines, &client.ListOptions{
		Namespace:     machineSetsNamespace,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	for _, machine := range machines.Items {
		subnetDesc, err := m.getDescriptorFromProviderSpec(machine.Spec.ProviderSpec.Value)
		if err != nil {
			return nil, err
		}
		subnetMap = append(subnetMap, *subnetDesc)
	}

	selector, _ = labels.Parse("machine.openshift.io/cluster-api-machine-role=master")
	machines = &machinev1beta1.MachineList{}
	err = m.client.List(ctx, machines, &client.ListOptions{
		Namespace:     machineSetsNamespace,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}
	for _, machine := range machines.Items {
		var subnetDesc *Subnet // declared here due to := rescoping of the masterResourceGroup variable below
		subnetDesc, err = m.getDescriptorFromProviderSpec(machine.Spec.ProviderSpec.Value)
		if err != nil {
			return nil, err
		}
		subnetDesc.IsMaster = true
		subnetMap = append(subnetMap, *subnetDesc)
	}

	return unique(subnetMap), nil
}

func (m *kubeManager) getDescriptorFromProviderSpec(providerSpec *kruntime.RawExtension) (*Subnet, error) {
	var spec machinev1beta1.AzureMachineProviderSpec
	err := json.Unmarshal(providerSpec.Raw, &spec)
	if err != nil {
		return nil, err
	}

	resource := azure.Resource{
		SubscriptionID: m.subscriptionID,
		ResourceGroup:  spec.NetworkResourceGroup,
		Provider:       "Microsoft.Network",
		ResourceType:   "virtualNetworks",
		ResourceName:   spec.Vnet,
	}

	return &Subnet{
		ResourceID: resource.String() + "/subnets/" + spec.Subnet,
	}, nil
}

func unique(s []Subnet) []Subnet {
	keys := make(map[string]struct{})
	list := []Subnet{}
	for _, entry := range s {
		key := strings.ToLower(entry.ResourceID)
		if _, ok := keys[key]; !ok {
			keys[key] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}
