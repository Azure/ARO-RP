package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/validate/dynamic"
)

/***************************************************************
	Monitor the Cluster Service Prinicpal required Permissions
****************************************************************/

func (mon *Monitor) emitValidatePermissions(ctx context.Context) error {
	subnets := []dynamic.Subnet{{
		ID:   mon.oc.Properties.MasterProfile.SubnetID,
		Path: "properties.masterProfile.subnetId",
	}}

	err := mon.validator.ValidateVnet(ctx, mon.oc.Location, subnets, mon.oc.Properties.NetworkProfile.PodCIDR,
		mon.oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		mon.emitGauge("cluster.validate.permissions", 1, map[string]string{
			"ValidateVnetPermissions": "Required permissions missing",
		})
	}

	err = mon.validator.ValidateSubnets(ctx, mon.oc, subnets)
	if err != nil {
		mon.emitGauge("cluster.validate.permissions", 1, map[string]string{
			"ValidateSubnet": "Required permissions Missing",
		})
	}

	err = mon.validator.ValidateDiskEncryptionSets(ctx, mon.oc)
	if err != nil {
		mon.emitGauge("cluster.validate.permissions", 1, map[string]string{
			"ValidateDiskEncryptionSet": "Required permissions Missing",
		})
	}
	return nil
}
