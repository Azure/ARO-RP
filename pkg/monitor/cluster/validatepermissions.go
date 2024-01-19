package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/validate/dynamic"
)

/***************************************************************
	Monitor the Cluster Service Prinicpal required permissions:
	- Network Contributor role on vnet
****************************************************************/

func (mon *Monitor) emitValidatePermissions(ctx context.Context) error {
	subnets := []dynamic.Subnet{{
		ID:   mon.oc.Properties.MasterProfile.SubnetID,
		Path: "properties.masterProfile.subnetId",
	}}

	err := mon.validator.ValidateVnet(ctx, mon.oc.Location, subnets, mon.oc.Properties.NetworkProfile.PodCIDR,
		mon.oc.Properties.NetworkProfile.ServiceCIDR)

	if err != nil {
		mon.emitGauge("cluster.validateVnet.permissions", 1, map[string]string{
			"vnetError": err.Error(),
		})
	}

	err = mon.validator.ValidateSubnets(ctx, mon.oc, subnets)
	if err != nil {
		mon.emitGauge("cluster.validateSubnets.permissions", 1, map[string]string{
			"subnetError": err.Error(),
		})
	}

	err = mon.validator.ValidateDiskEncryptionSets(ctx, mon.oc)
	if err != nil {
		mon.emitGauge("cluster.validateDiskEncryptionSets.permissions", 1, map[string]string{
			"diskEncryptionSetError": err.Error(),
		})
	}
	return nil
}
