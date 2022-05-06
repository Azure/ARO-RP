package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
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
		Machines: make([]MachinesInformation, 0, len(machines.Items)),
	}

	for _, machine := range machines.Items {
		lastOperation := "Unknown"
		if machine.Status.LastOperation != nil &&
			machine.Status.LastOperation.Description != nil {
			lastOperation = *machine.Status.LastOperation.Description
		}

		lastOperationDate := "Unknown"
		if machine.Status.LastOperation != nil &&
			machine.Status.LastOperation.LastUpdated != nil {
			lastOperationDate = machine.Status.LastOperation.LastUpdated.String()
		}

		status := "Unknown"
		if machine.Status.Phase != nil {
			status = *machine.Status.Phase
		}

		errorReason := "None"
		if machine.Status.ErrorReason != nil {
			errorReason = string(*machine.Status.ErrorReason)
		}

		errorMessage := "None"
		if machine.Status.ErrorMessage != nil {
			errorMessage = *machine.Status.ErrorMessage
		}

		final.Machines = append(final.Machines, MachinesInformation{
			Name:              machine.Name,
			CreatedTime:       machine.CreationTimestamp.String(),
			LastUpdated:       machine.Status.LastUpdated.String(),
			ErrorReason:       errorReason,
			ErrorMessage:      errorMessage,
			LastOperation:     lastOperation,
			LastOperationDate: lastOperationDate,
			Status:            status,
		})
	}

	return final
}

func (f *realFetcher) Machines(ctx context.Context) (*MachineListInformation, error) {
	r, err := f.machineclient.MachineV1beta1().Machines("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return MachinesFromMachineList(r), nil
}

func (c *client) Machines(ctx context.Context) (*MachineListInformation, error) {
	return c.fetcher.Machines(ctx)
}
