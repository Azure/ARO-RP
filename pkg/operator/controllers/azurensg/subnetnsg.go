package azurensg

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type subnetDescriptor struct {
	resourceGroup string
	vnetName      string
	subnetName    string
}

func (r *Reconciler) reconcileSubnetNSG(ctx context.Context, instance *arov1alpha1.Cluster, subscriptionID string, subnetsClient network.SubnetsClient) error {
	// the main logic starts here
	subnets, masterResourceGroup, err := r.getSubnets(ctx)
	if err != nil {
		return err
	}
	for ws := range subnets {
		err = r.ensureSubnetNSG(ctx, subnetsClient, subscriptionID, masterResourceGroup, instance.Spec.InfraID, api.ArchitectureVersion(instance.Spec.ArchitectureVersion), ws.resourceGroup, ws.vnetName, ws.subnetName, subnets[ws])
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) ensureSubnetNSG(ctx context.Context, subnetsClient network.SubnetsClient, subscriptionID, resourcesResourceGroup, infraID string, architectureVersion api.ArchitectureVersion, vnetResourceGroup, vnetName, subnetName string, isWorkerSubnet bool) error {
	subnetObject, err := subnetsClient.Get(ctx, vnetResourceGroup, vnetName, subnetName, "")
	if err != nil {
		return err
	}
	if subnetObject.SubnetPropertiesFormat == nil || subnetObject.SubnetPropertiesFormat.NetworkSecurityGroup == nil {
		return fmt.Errorf("received nil, expected a value in SubnetProperties when trying to Get subnet %s/%s in resource group %s", vnetName, subnetName, vnetResourceGroup)
	}
	correctNSGResourceID, err := subnet.NetworkSecurityGroupIDExpanded(architectureVersion, resourcesResourceGroup, infraID, isWorkerSubnet)
	if err != nil {
		return err
	}
	correctNSGResourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionID, correctNSGResourceID)

	if !strings.EqualFold(*subnetObject.NetworkSecurityGroup.ID, correctNSGResourceID) {
		r.log.Infof("Fixing NSG from %s to %s", *subnetObject.NetworkSecurityGroup.ID, correctNSGResourceID)
		// NSG doesn't match - fixing
		subnetObject.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{ID: &correctNSGResourceID}
		err = subnetsClient.CreateOrUpdateAndWait(ctx, vnetResourceGroup, vnetName, subnetName, subnetObject)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) getSubnets(ctx context.Context) (map[subnetDescriptor]bool, string, error) {
	subnetMap := make(map[subnetDescriptor]bool) // bool is true for worker subnets
	var masterResourceGroup *string
	// select all workers by the  machine.openshift.io/cluster-api-machine-role: not equal to master Label
	machines, err := r.maocli.MachineV1beta1().Machines(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role!=master"})
	if err != nil {
		return nil, "", err
	}

	for _, machine := range machines.Items {
		_, subnetDesc, err := r.getDescriptorFromProviderSpec(machine.Spec.ProviderSpec.Value)
		if err != nil {
			return nil, "", err
		}

		// subnetMap stores boolean isWorker
		subnetMap[*subnetDesc] = true
	}
	// select all masters
	machines, err = r.maocli.MachineV1beta1().Machines(machineSetsNamespace).List(ctx, metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machine-role=master"})
	if err != nil {
		return nil, "", err
	}
	for _, machine := range machines.Items {
		var subnetDesc *subnetDescriptor // declared here due to := rescoping of the masterResourceGroup variable below
		masterResourceGroup, subnetDesc, err = r.getDescriptorFromProviderSpec(machine.Spec.ProviderSpec.Value)
		if err != nil {
			return nil, "", err
		}

		// subnetMap stores boolean isWorker
		subnetMap[*subnetDesc] = false
	}
	if masterResourceGroup == nil {
		return nil, "", fmt.Errorf("master resource group not found")
	}
	return subnetMap, *masterResourceGroup, nil
}

func (r *Reconciler) getDescriptorFromProviderSpec(providerSpec *runtime.RawExtension) (*string, *subnetDescriptor, error) {
	var spec azureproviderv1beta1.AzureMachineProviderSpec
	err := json.Unmarshal(providerSpec.Raw, &spec)
	if err != nil {
		return nil, nil, err
	}
	return &spec.ResourceGroup, &subnetDescriptor{
		resourceGroup: spec.NetworkResourceGroup,
		vnetName:      spec.Vnet,
		subnetName:    spec.Subnet,
	}, nil
}
