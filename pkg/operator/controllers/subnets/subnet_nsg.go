package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

const (
	AnnotationTimestamp = "aro.openshift.io/lastSubnetReconcileTimestamp"
)

func (r *reconcileManager) ensureSubnetNSG(ctx context.Context, s subnet.Subnet) error {
	architectureVersion := api.ArchitectureVersion(r.instance.Spec.ArchitectureVersion)

	subnetObject, err := r.subnets.Get(ctx, s.ResourceID)
	if err != nil {
		if azureerrors.IsNotFoundError(err) {
			r.log.Infof("Subnet %s not found, skipping", s.ResourceID)
			return nil
		}
		return err
	}
	if subnetObject.SubnetPropertiesFormat == nil {
		return fmt.Errorf("received nil, expected a value in subnetProperties when trying to Get subnet %s", s.ResourceID)
	}

	correctNSGResourceID, err := apisubnet.NetworkSecurityGroupIDExpanded(architectureVersion, r.instance.Spec.ClusterResourceGroupID, r.instance.Spec.InfraID, !s.IsMaster)
	if err != nil {
		return err
	}

	// if the NSG is assigned && it's the correct NSG - do nothing
	if subnetObject.SubnetPropertiesFormat.NetworkSecurityGroup != nil && strings.EqualFold(*subnetObject.NetworkSecurityGroup.ID, correctNSGResourceID) {
		return nil
	}

	// else the NSG assignment needs to be corrected
	oldNSG := "nil"
	if subnetObject.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
		oldNSG = *subnetObject.NetworkSecurityGroup.ID
	}
	r.log.Infof("Fixing NSG from %s to %s", oldNSG, correctNSGResourceID)
	subnetObject.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{ID: &correctNSGResourceID}
	err = r.subnets.CreateOrUpdate(ctx, s.ResourceID, subnetObject)
	if err != nil {
		return err
	}

	return r.updateReconcileSubnetAnnotation(ctx)
}

func (r *reconcileManager) updateReconcileSubnetAnnotation(ctx context.Context) error {
	if r.instance.Annotations == nil {
		r.instance.Annotations = make(map[string]string)
	}
	r.instance.Annotations[AnnotationTimestamp] = time.Now().Format(time.RFC1123)
	return r.client.Update(ctx, r.instance)
}
