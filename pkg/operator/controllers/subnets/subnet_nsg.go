package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type subnetDescriptor struct {
	resourceID string
	isMaster   bool
}

func (r *reconcileManager) reconcileSubnets(ctx context.Context) error {
	// the main logic starts here
	subnets, err := r.getSubnets(ctx)
	if err != nil {
		return err
	}

	for _, s := range subnets {
		err = r.ensureSubnetNSG(ctx, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *reconcileManager) ensureSubnetNSG(ctx context.Context, s subnetDescriptor) error {
	architectureVersion := api.ArchitectureVersion(r.instance.Spec.ArchitectureVersion)

	subnetObject, err := r.manager.Get(ctx, s.resourceID)
	if err != nil {
		return err
	}
	if subnetObject.SubnetPropertiesFormat == nil || subnetObject.SubnetPropertiesFormat.NetworkSecurityGroup == nil {
		return fmt.Errorf("received nil, expected a value in subnetProperties when trying to Get subnet %s", s.resourceID)
	}

	correctNSGResourceID, err := subnet.NetworkSecurityGroupIDExpanded(architectureVersion, r.instance.Spec.ClusterResourceGroupID, r.instance.Spec.InfraID, !s.isMaster)
	if err != nil {
		return err
	}

	if !strings.EqualFold(*subnetObject.NetworkSecurityGroup.ID, correctNSGResourceID) {
		r.log.Infof("Fixing NSG from %s to %s", *subnetObject.NetworkSecurityGroup.ID, correctNSGResourceID)
		// NSG doesn't match - fixing
		subnetObject.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{ID: &correctNSGResourceID}
		err = r.manager.CreateOrUpdate(ctx, s.resourceID, subnetObject)
		if err != nil {
			return err
		}
	}
	return nil
}

// getSubnets reconstructs subnetId used in machines
// Example : /subscriptions/{subscriptionID}/resourceGroups/{vnet-resource-group}/providers/Microsoft.Network/virtualNetworks/{vnet-name}/subnets/{subnet-name}
func (r *reconcileManager) getSubnets(ctx context.Context) ([]subnetDescriptor, error) {
	subnetMap := []subnetDescriptor{}

	// select all workers by the  machine.openshift.io/cluster-api-machine-role: not equal to master Label
	machines, err := r.maocli.MachineV1beta1().Machines(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role!=master"})
	if err != nil {
		return nil, err
	}

	for _, machine := range machines.Items {
		subnetDesc, err := r.getDescriptorFromProviderSpec(machine.Spec.ProviderSpec.Value)
		if err != nil {
			return nil, err
		}
		subnetMap = append(subnetMap, *subnetDesc)
	}
	machines, err = r.maocli.MachineV1beta1().Machines(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role=master"})
	if err != nil {
		return nil, err
	}
	for _, machine := range machines.Items {
		var subnetDesc *subnetDescriptor // declared here due to := rescoping of the masterResourceGroup variable below
		subnetDesc, err = r.getDescriptorFromProviderSpec(machine.Spec.ProviderSpec.Value)
		if err != nil {
			return nil, err
		}
		subnetDesc.isMaster = true
		subnetMap = append(subnetMap, *subnetDesc)
	}

	return subnetMap, nil
}

func (r *reconcileManager) getDescriptorFromProviderSpec(providerSpec *runtime.RawExtension) (*subnetDescriptor, error) {
	var spec azureproviderv1beta1.AzureMachineProviderSpec
	err := json.Unmarshal(providerSpec.Raw, &spec)
	if err != nil {
		return nil, err
	}

	resource := azure.Resource{
		SubscriptionID: r.subscriptionID,
		ResourceGroup:  spec.NetworkResourceGroup,
		Provider:       "Microsoft.Network",
		ResourceType:   "virtualNetworks",
		ResourceName:   spec.Vnet,
	}

	return &subnetDescriptor{
		resourceID: resource.String() + "/subnets/" + spec.Subnet,
	}, nil
}
