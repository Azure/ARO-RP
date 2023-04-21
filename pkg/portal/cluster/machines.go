package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MachinesInformation struct {
	Name              string `json:"name"`
	CreatedTime       string `json:"createdTime"`
	LastUpdated       string `json:"lastUpdated"`
	ErrorReason       string `json:"errorReason"`
	ErrorMessage      string `json:"errorMessage"`
	LastOperation     string `json:"lastOperation"`
	LastOperationDate string `json:"lastOperationDate"`
	Status            string `json:"status"`
}

type MachineListInformation struct {
	Machines []MachinesInformation `json:"machines"`
}

func MachinesFromMachineList(machines *machinev1beta1.MachineList) *MachineListInformation {
	final := &MachineListInformation{
		Machines: make([]MachinesInformation, len(machines.Items)),
	}

	for i, machine := range machines.Items {
		machinesInformation := MachinesInformation{
			Name:              machine.Name,
			CreatedTime:       machine.CreationTimestamp.String(),
			LastUpdated:       machine.Status.LastUpdated.String(),
			ErrorReason:       getErrorReason(machine),
			ErrorMessage:      getErrorMessage(machine),
			LastOperation:     getLastOperation(machine),
			LastOperationDate: getLastOperationDate(machine),
			Status:            getStatus(machine),
		}
		final.Machines[i] = machinesInformation
	}
	return final
}

func (f *realFetcher) Machines(ctx context.Context) (*MachineListInformation, error) {
	r, err := f.machineClient.MachineV1beta1().Machines("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return MachinesFromMachineList(r), nil
}

func (c *client) Machines(ctx context.Context) (*MachineListInformation, error) {
	return c.fetcher.Machines(ctx)
}

func (c *azureClient) VMAllocationStatus(ctx context.Context) (map[string]string, error) {
	return c.fetcher.vmAllocationStatus(ctx)
}

func (f *azureSideFetcher) vmAllocationStatus(ctx context.Context) (map[string]string, error) {
	subscriptionDoc := f.subscriptionDoc
	clusterRGName := f.resourceGroupName
	aroEnvironment := f.env.Environment()
	spAuthorizer := f.spAuthorizer
	computeResources, err := f.resourceClientFactory.NewResourcesClient(aroEnvironment, subscriptionDoc.ID, spAuthorizer).ListByResourceGroup(ctx, clusterRGName, "resourceType eq 'Microsoft.Compute/virtualMachines'", "", nil)
	if err != nil {
		return nil, err
	}
	vmAllocationStatus := make(map[string]string)
	virtualMachineClient := f.virtualMachinesClientFactory.NewVirtualMachinesClient(aroEnvironment, subscriptionDoc.ID, spAuthorizer)
	for _, res := range computeResources {
		putAllocationStatusToMap(ctx, clusterRGName, vmAllocationStatus, res, virtualMachineClient, f.log)
	}

	return vmAllocationStatus, nil
}

// Helper Functions
func putAllocationStatusToMap(ctx context.Context, clusterRGName string, vmAllocationStatus map[string]string, res mgmtfeatures.GenericResourceExpanded, virtualMachineClient compute.VirtualMachinesClient, log *logrus.Entry) {
	vm, err := virtualMachineClient.Get(ctx, clusterRGName, *res.Name, mgmtcompute.InstanceView)
	if err != nil {
		log.Warn(err) // can happen when the ARM cache is lagging
		return
	}

	vmName := *vm.Name
	instanceViewStatuses := vm.InstanceView.Statuses
	for _, status := range *instanceViewStatuses {
		if strings.HasPrefix(*status.Code, "PowerState/") {
			vmAllocationStatus[vmName] = *status.Code
			return
		}
	}
	vmAllocationStatus[vmName] = ""
}

func getLastOperation(machine machinev1beta1.Machine) string {
	lastOperation := "Unknown"
	if machine.Status.LastOperation != nil &&
		machine.Status.LastOperation.Description != nil {
		lastOperation = *machine.Status.LastOperation.Description
	}
	return lastOperation
}

func getLastOperationDate(machine machinev1beta1.Machine) string {
	lastOperationDate := "Unknown"
	if machine.Status.LastOperation != nil &&
		machine.Status.LastOperation.LastUpdated != nil {
		lastOperationDate = machine.Status.LastOperation.LastUpdated.String()
	}
	return lastOperationDate
}

func getStatus(machine machinev1beta1.Machine) string {
	status := "Unknown"
	if machine.Status.Phase != nil {
		status = *machine.Status.Phase
	}
	return status
}

func getErrorReason(machine machinev1beta1.Machine) string {
	errorReason := "None"
	if machine.Status.ErrorReason != nil {
		errorReason = string(*machine.Status.ErrorReason)
	}
	return errorReason
}

func getErrorMessage(machine machinev1beta1.Machine) string {
	errorMessage := "None"
	if machine.Status.ErrorMessage != nil {
		errorMessage = *machine.Status.ErrorMessage
	}
	return errorMessage
}
