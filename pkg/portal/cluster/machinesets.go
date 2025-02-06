package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

type OsDiskManagedDisk struct {
	StorageAccountType string `json:"storageaccounttype"`
}
type MachineSetProviderSpecValueOSDisk struct {
	DiskSizeGB  int               `json:"disksizegb"`
	OsType      string            `json:"ostype"`
	ManagedDisk OsDiskManagedDisk `json:"manageddisk"`
}
type MachineSetProviderSpecValue struct {
	Kind                 string                            `json:"kind"`
	Location             string                            `json:"location"`
	NetworkResourceGroup string                            `json:"networkresourcegroup"`
	OsDisk               MachineSetProviderSpecValueOSDisk `json:"osdisk"`
	PublicIP             bool                              `json:"publicip"`
	PublicLoadBalancer   string                            `json:"publicloadbalancer"`
	Subnet               string                            `json:"subnet"`
	VmSize               string                            `json:"vmsize"`
	Vnet                 string                            `json:"vnet"`
}

type MachineSetsInformation struct {
	Name                     string `json:"name"`
	Type                     string `json:"type"`
	CreatedAt                string `json:"createdat"`
	DesiredReplicas          int    `json:"desiredreplicas"`
	Replicas                 int    `json:"replicas"`
	ErrorReason              string `json:"errorreason"`
	ErrorMessage             string `json:"errormessage"`
	PublicLoadBalancerName   string `json:"publicloadbalancername"`
	VMSize                   string `json:"vmsize"`
	OSDiskAccountStorageType string `json:"accountstoragetype"`
	Subnet                   string `json:"subnet"`
	VNet                     string `json:"vnet"`
}
type MachineSetListInformation struct {
	MachineSets []MachineSetsInformation `json:"machines"`
}

func (f *realFetcher) MachineSetsFromMachineSetList(ctx context.Context, machineSets *machinev1beta1.MachineSetList) *MachineSetListInformation {
	final := &MachineSetListInformation{
		MachineSets: make([]MachineSetsInformation, 0, len(machineSets.Items)),
	}

	for _, machineSet := range machineSets.Items {
		var machineSetProviderSpecValue MachineSetProviderSpecValue
		machineSetJson, err := machineSet.Spec.Template.Spec.ProviderSpec.Value.MarshalJSON()
		if err != nil {
			f.log.Logger.Error(err.Error())
		}
		json.Unmarshal(machineSetJson, &machineSetProviderSpecValue)

		final.MachineSets = append(final.MachineSets, MachineSetsInformation{
			Name:                     machineSet.Name,
			Type:                     machineSet.ObjectMeta.Labels["machine.openshift.io/cluster-api-machine-type"],
			CreatedAt:                machineSet.ObjectMeta.CreationTimestamp.String(),
			DesiredReplicas:          int(*machineSet.Spec.Replicas),
			Replicas:                 int(machineSet.Status.Replicas),
			ErrorReason:              getErrorReasonMachineSet(machineSet),
			ErrorMessage:             getErrorMessageMachineSet(machineSet),
			PublicLoadBalancerName:   machineSetProviderSpecValue.PublicLoadBalancer,
			OSDiskAccountStorageType: machineSetProviderSpecValue.OsDisk.ManagedDisk.StorageAccountType,
			VNet:                     machineSetProviderSpecValue.Vnet,
			Subnet:                   machineSetProviderSpecValue.Subnet,
			VMSize:                   machineSetProviderSpecValue.VmSize,
		})
	}

	return final
}

func (f *realFetcher) MachineSets(ctx context.Context) (*MachineSetListInformation, error) {
	r, err := f.machineClient.MachineV1beta1().MachineSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return f.MachineSetsFromMachineSetList(ctx, r), nil
}

func (c *client) MachineSets(ctx context.Context) (*MachineSetListInformation, error) {
	return c.fetcher.MachineSets(ctx)
}

// Helper functions
func getErrorMessageMachineSet(machineSet machinev1beta1.MachineSet) string {
	errorMessage := "None"
	if machineSet.Status.ErrorMessage != nil {
		errorMessage = *machineSet.Status.ErrorMessage
	}
	return errorMessage
}

func getErrorReasonMachineSet(machineSet machinev1beta1.MachineSet) string {
	errorReason := "None"
	if machineSet.Status.ErrorReason != nil {
		errorReason = string(*machineSet.Status.ErrorReason)
	}
	return errorReason
}
