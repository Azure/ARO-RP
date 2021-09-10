package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (r *reconcileManager) ensureSubnetNSG(ctx context.Context, s subnet.Subnet) error {
	architectureVersion := api.ArchitectureVersion(r.instance.Spec.ArchitectureVersion)

	subnetObject, err := r.subnets.Get(ctx, s.ResourceID)
	if err != nil {
		return err
	}
	if subnetObject.SubnetPropertiesFormat == nil || subnetObject.SubnetPropertiesFormat.NetworkSecurityGroup == nil {
		return fmt.Errorf("received nil, expected a value in subnetProperties when trying to Get subnet %s", s.ResourceID)
	}

	correctNSGResourceID, err := subnet.NetworkSecurityGroupIDExpanded(architectureVersion, r.instance.Spec.ClusterResourceGroupID, r.instance.Spec.InfraID, !s.IsMaster)
	if err != nil {
		return err
	}

	if !strings.EqualFold(*subnetObject.NetworkSecurityGroup.ID, correctNSGResourceID) {
		r.log.Infof("Fixing NSG from %s to %s", *subnetObject.NetworkSecurityGroup.ID, correctNSGResourceID)

		subnetObject.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{ID: &correctNSGResourceID}
		err = r.subnets.CreateOrUpdate(ctx, s.ResourceID, subnetObject)
		if err != nil {
			return err
		}
	}
	return nil
}
