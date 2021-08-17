package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
)

// KubeManager interface interact with kubernetes layer to extract required information
type KubeManager interface {
	ListFromCluster(ctx context.Context) ([]Subnet, error)
}

type kubeManager struct {
	maocli maoclient.Interface

	subscriptionID string
}

func NewKubeManager(maocli maoclient.Interface, subscriptionID string) KubeManager {
	return &kubeManager{
		maocli:         maocli,
		subscriptionID: subscriptionID,
	}
}

// ListFromCluster reconstructs subnetId used in machines object in the cluster
// In cases we interat with customer vnets, we don't know which subnets are used in ARO.
// Example : /subscriptions/{subscriptionID}/resourceGroups/{vnet-resource-group}/providers/Microsoft.Network/virtualNetworks/{vnet-name}/subnets/{subnet-name}
func (m *kubeManager) ListFromCluster(ctx context.Context) ([]Subnet, error) {
	if m.maocli == nil {
		return nil, fmt.Errorf("subnets.Manager was initiated without kubernetes client")
	}
	subnetMap := []Subnet{}

	// select all workers by the  machine.openshift.io/cluster-api-machine-role: not equal to master Label
	machines, err := m.maocli.MachineV1beta1().Machines(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role!=master"})
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
	machines, err = m.maocli.MachineV1beta1().Machines(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role=master"})
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

func (m *kubeManager) getDescriptorFromProviderSpec(providerSpec *runtime.RawExtension) (*Subnet, error) {
	var spec azureproviderv1beta1.AzureMachineProviderSpec
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
	keys := make(map[string]bool)
	list := []Subnet{}
	for _, entry := range s {
		if _, value := keys[entry.ResourceID]; !value {
			keys[entry.ResourceID] = true
			list = append(list, entry)
		}
	}
	return list
}
