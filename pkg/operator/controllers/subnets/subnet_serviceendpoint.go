package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (r *reconcileManager) ensureSubnetServiceEndpoints(ctx context.Context, s subnet.Subnet) error {
	if !operator.GatewayEnabled(r.instance) {
		r.log.Debug("Reconciling service endpoints on subnet ", s.ResourceID)

		subnetID, err := arm.ParseResourceID(s.ResourceID)
		if err != nil {
			return err
		}
		subnetObject, err := r.subnets.Get(ctx, subnetID.ResourceGroupName, subnetID.Parent.Name, subnetID.Name, nil)
		if err != nil {
			if azureerrors.IsNotFoundError(err) {
				r.log.Infof("Subnet %s not found, skipping. err: %v", s.ResourceID, err)
				return nil
			}
			return err
		}
		if r.subnets == nil { // just in case
			return fmt.Errorf("subnet can't be nil")
		}

		var changed bool
		if subnetObject.Properties == nil {
			subnetObject.Properties = &armnetwork.SubnetPropertiesFormat{}
		}
		if subnetObject.Properties.ServiceEndpoints == nil {
			subnetObject.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{}
		}

		for _, endpoint := range api.SubnetsEndpoints {
			var found bool
			for _, se := range subnetObject.Properties.ServiceEndpoints {
				if strings.EqualFold(*se.Service, endpoint) &&
					*se.ProvisioningState == armnetwork.ProvisioningStateSucceeded {
					found = true
				}
			}
			if !found {
				subnetObject.Properties.ServiceEndpoints = append(subnetObject.Properties.ServiceEndpoints, &armnetwork.ServiceEndpointPropertiesFormat{
					Service:   to.StringPtr(endpoint),
					Locations: []*string{to.StringPtr("*")},
				})
				changed = true
			}
		}

		if changed {
			err = r.subnets.CreateOrUpdateAndWait(ctx, subnetID.ResourceGroupName, subnetID.Parent.Name, subnetID.Name, subnetObject.Subnet, nil)
			if err != nil {
				return err
			}
		}
		return nil
	}

	r.log.Debug("Skipping service endpoint reconciliation since egress lockdown is enabled")
	return nil
}
