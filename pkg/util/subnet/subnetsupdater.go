package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

type Updater interface {
	AddEndpointsToSubnets(endpoints []string, subnets []*mgmtnetwork.Subnet) (updatedSubnets []*mgmtnetwork.Subnet)
}

type UpdaterManager struct {
}

func (um *UpdaterManager) AddEndpointsToSubnets(endpoints []string, subnets []*mgmtnetwork.Subnet) (updatedSubnets []*mgmtnetwork.Subnet) {
	for _, subnet := range subnets {
		if subnetChanged := addEndpointsToSubnet(endpoints, subnet); subnetChanged {
			updatedSubnets = append(updatedSubnets, subnet)
		}
	}

	return updatedSubnets
}

func addEndpointsToSubnet(endpoints []string, subnet *mgmtnetwork.Subnet) (subnetChanged bool) {
	for _, endpoint := range endpoints {
		endpointFound, serviceEndpointPtr := subnetContainsEndpoint(subnet, endpoint)
		endpointSucceded := serviceEndpointPtr != nil && serviceEndpointPtr.ProvisioningState == mgmtnetwork.Succeeded

		if !endpointFound || !endpointSucceded {
			addEndpointToSubnet(endpoint, subnet)
			subnetChanged = true
		}
	}

	return subnetChanged
}

func subnetContainsEndpoint(subnet *mgmtnetwork.Subnet, endpoint string) (endpointFound bool, serviceEndpointPtr *mgmtnetwork.ServiceEndpointPropertiesFormat) {
	if subnet == nil || subnet.ServiceEndpoints == nil {
		return false, nil
	}

	for _, serviceEndpoint := range *subnet.ServiceEndpoints {
		if endpointFound = strings.EqualFold(*serviceEndpoint.Service, endpoint); endpointFound {
			return true, &serviceEndpoint
		}
	}

	return false, nil
}

func addEndpointToSubnet(endpoint string, subnet *mgmtnetwork.Subnet) {
	if subnet.ServiceEndpoints == nil {
		subnet.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{}
	}

	serviceEndpoint := mgmtnetwork.ServiceEndpointPropertiesFormat{
		Service:   to.StringPtr(endpoint),
		Locations: &[]string{"*"},
	}

	*subnet.ServiceEndpoints = append(*subnet.ServiceEndpoints, serviceEndpoint)
}
