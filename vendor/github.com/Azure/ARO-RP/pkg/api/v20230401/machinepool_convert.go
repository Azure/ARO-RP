package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type machinePoolConverter struct{}

func (c machinePoolConverter) ToExternal(mp *api.MachinePool) interface{} {
	out := new(MachinePool)
	out.proxyResource = true
	out.ID = mp.ID
	out.Name = mp.Name
	out.Type = mp.Type
	out.Properties.Resources = mp.Properties.Resources
	return out
}

func (c machinePoolConverter) ToInternal(_mp interface{}, out *api.MachinePool) {
	ocm := _mp.(*api.MachinePool)
	out.ID = ocm.ID
}

// ToExternalList returns a slice of external representations of the internal objects
func (c machinePoolConverter) ToExternalList(mp []*api.MachinePool) interface{} {
	l := &MachinePoolList{
		MachinePools: make([]*MachinePool, 0, len(mp)),
	}

	for _, machinepool := range mp {
		c := c.ToExternal(machinepool)
		l.MachinePools = append(l.MachinePools, c.(*MachinePool))
	}

	return l
}
