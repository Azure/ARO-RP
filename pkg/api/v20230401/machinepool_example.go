package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleMachinePool() *MachinePool {
	doc := api.ExampleClusterManagerConfigurationDocumentMachinePool()
	ext := (&machinePoolConverter{}).ToExternal(doc.MachinePool)
	return ext.(*MachinePool)
}

func ExampleMachinePoolPutParameter() interface{} {
	mp := exampleMachinePool()
	mp.ID = ""
	mp.Type = ""
	mp.Name = ""
	return mp
}

func ExampleMachinePoolPatchParameter() interface{} {
	return ExampleMachinePoolPutParameter()
}

func ExampleMachinePoolResponse() interface{} {
	return exampleMachinePool()
}

func ExampleMachinePoolListResponse() interface{} {
	return &MachinePoolList{
		MachinePools: []*MachinePool{
			ExampleMachinePoolResponse().(*MachinePool),
		},
	}
}
