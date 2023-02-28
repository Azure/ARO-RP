package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
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

func (c *client) VMAllocationStatus(ctx context.Context) (map[string]string, error) {
	return c.fetcher.VMAllocationStatus(ctx)
}

func (f *realFetcher) VMAllocationStatus(ctx context.Context) (map[string]string, error) {
	env := f.azureSideFetcher.env
	subscriptionDoc := f.azureSideFetcher.subscriptionDoc
	clusterRGName := f.azureSideFetcher.resourceGroupName
	fpAuth, err := env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}
	// Getting Virtual Machine resources through the Cluster's Resource Group
	computeResources, err := f.resourceFactory.NewResourcesClient(env.Environment(), subscriptionDoc.ID, fpAuth).ListByResourceGroup(ctx, clusterRGName, "resourceType eq 'Microsoft.Compute/virtualMachines'", "", nil)
	if err != nil {
		return nil, err
	}
	vmAllocationStatus := make(map[string]string)
	virtualMachineClient := f.resourceFactory.NewVirtualMachinesClient(env.Environment(), subscriptionDoc.ID, fpAuth)
	for _, res := range computeResources {
		putAllocationStatusToMap(ctx, clusterRGName, vmAllocationStatus, res, virtualMachineClient, f.log)
	}

	return vmAllocationStatus, nil
}

// Helper Functions
func putAllocationStatusToMap(ctx context.Context, clusterRGName string, vmAllocationStatus map[string]string, res mgmtfeatures.GenericResourceExpanded, virtualMachineClient compute.VirtualMachinesClient, log *logrus.Entry) {
	var vmName, allocationStatus string
	vm, err := virtualMachineClient.Get(ctx, clusterRGName, *res.Name, mgmtcompute.InstanceView)
	if err != nil {
		log.Warn(err) // can happen when the ARM cache is lagging
		return
	}

	vmName = *vm.Name
	instanceViewStatuses := vm.InstanceView.Statuses
	for _, status := range *instanceViewStatuses {
		if strings.HasPrefix(*status.Code, "PowerState/") {
			allocationStatus = *status.Code
		}
	}

	vmAllocationStatus[vmName] = allocationStatus
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
