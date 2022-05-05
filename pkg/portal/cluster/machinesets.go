package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	machineapi "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MachineSetsInformation struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	CreatedAt       string `json:"createdat"`
	DesiredReplicas int    `json:"desiredreplicas"`
	Replicas        int    `json:"replicas"`
	ErrorReason     string `json:"errorreason"`
	ErrorMessage    string `json:"errormessage"`
}

type MachineSetListInformation struct {
	MachineSets []MachineSetsInformation `json:"machines"`
}

func MachineSetsFromMachineSetList(machineSets *machineapi.MachineSetList) *MachineSetListInformation {
	final := &MachineSetListInformation{
		MachineSets: make([]MachineSetsInformation, 0, len(machineSets.Items)),
	}

	for _, machineSet := range machineSets.Items {

		errorReason := "None"
		if machineSet.Status.ErrorReason != nil {
			errorReason = string(*machineSet.Status.ErrorReason)
		}

		errorMessage := "None"
		if machineSet.Status.ErrorMessage != nil {
			errorMessage = *machineSet.Status.ErrorMessage
		}
		final.MachineSets = append(final.MachineSets, MachineSetsInformation{
			Name:            machineSet.Name,
			Type:            machineSet.ObjectMeta.Labels["machine.openshift.io/cluster-api-machine-type"],
			CreatedAt:       machineSet.ObjectMeta.CreationTimestamp.String(),
			DesiredReplicas: int(*machineSet.Spec.Replicas),
			Replicas:        int(machineSet.Status.Replicas),
			ErrorReason:     errorReason,
			ErrorMessage:    errorMessage,
		})
	}

	return final
}

func (f *realFetcher) MachineSets(ctx context.Context) (*MachineSetListInformation, error) {
	r, err := f.maoclient.MachineV1beta1().MachineSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return MachineSetsFromMachineSetList(r), nil
}

func (c *client) MachineSets(ctx context.Context) (*MachineSetListInformation, error) {
	return c.fetcher.MachineSets(ctx)
}
