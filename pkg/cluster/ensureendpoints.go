package cluster

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
)

// ensureServiceEndpoints should enable service endpoints on
// subnets for storage account access, but only if egress lockdown is
// not enabled.
func (m *manager) ensureServiceEndpoints(ctx context.Context) error {
	// Only add service endpoints to the subnet if egress lockdown is not enabled.
	if m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled {
		return nil
	}

	subnetIds, err := m.getSubnetIds()
	if err != nil {
		return err
	}

	for _, subnetId := range subnetIds {
		r, err := arm.ParseResourceID(subnetId)
		if err != nil {
			return err
		}
		subnet, err := m.armSubnets.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, nil)
		if err != nil {
			return err
		}
		shouldUpdate := addEndpointsToSubnet(api.SubnetsEndpoints, &subnet.Subnet)
		if !shouldUpdate {
			continue
		}
		err = m.armSubnets.CreateOrUpdateAndWait(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, subnet.Subnet, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *manager) getSubnetIds() ([]string, error) {
	subnets := []string{
		m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
	}
	workerProfiles, _ := api.GetEnrichedWorkerProfiles(m.doc.OpenShiftCluster.Properties)

	for _, wp := range workerProfiles {
		if len(wp.SubnetID) == 0 {
			return nil, fmt.Errorf("WorkerProfile '%s' has no SubnetID; check that the corresponding MachineSet is valid", wp.Name)
		}
		subnets = append(subnets, wp.SubnetID)
	}
	return subnets, nil
}

// addEndpointsToSubnet adds the endpoints (that either are missing in subnet
// or aren't in succeeded state in the subnet) to the subnet and returns the updated subnet
func addEndpointsToSubnet(endpoints []string, subnet *armnetwork.Subnet) (subnetChanged bool) {
	for _, endpoint := range endpoints {
		endpointFound, serviceEndpointPtr := subnetContainsEndpoint(subnet, endpoint)

		if !endpointFound || *serviceEndpointPtr.ProvisioningState != armnetwork.ProvisioningStateSucceeded {
			addEndpointToSubnet(endpoint, subnet)
			subnetChanged = true
		}
	}

	return subnetChanged
}

// subnetContainsEndpoint returns false and nil if subnet does not contain the endpoint.
// If the subnet does contain the endpoint, true and a pointer to the service endpoint
// is returned to be able to do additional checks and perform actions accordingly.
func subnetContainsEndpoint(subnet *armnetwork.Subnet, endpoint string) (endpointFound bool, serviceEndpointPtr *armnetwork.ServiceEndpointPropertiesFormat) {
	if subnet == nil || subnet.Properties.ServiceEndpoints == nil {
		return false, nil
	}

	for _, serviceEndpoint := range subnet.Properties.ServiceEndpoints {
		if endpointFound = strings.EqualFold(*serviceEndpoint.Service, endpoint); endpointFound {
			return true, serviceEndpoint
		}
	}

	return false, nil
}

// addEndpointToSubnet appends the endpoint to the slice of ServiceEndpoints of the subnet.
func addEndpointToSubnet(endpoint string, subnet *armnetwork.Subnet) {
	if subnet.Properties.ServiceEndpoints == nil {
		subnet.Properties.ServiceEndpoints = []*armnetwork.ServiceEndpointPropertiesFormat{}
	}

	serviceEndpoint := armnetwork.ServiceEndpointPropertiesFormat{
		Service:   to.StringPtr(endpoint),
		Locations: []*string{to.StringPtr("*")},
	}

	subnet.Properties.ServiceEndpoints = append(subnet.Properties.ServiceEndpoints, &serviceEndpoint)
}
