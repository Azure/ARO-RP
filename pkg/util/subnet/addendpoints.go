package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

// addEndpointsToSubnets adds the endpoints (that either are missing in subnets
// or aren't in succeded state in subnets) to the subnets and returns those updated subnets.
// This method does not talk to any external dependecies to remain pure bussiness logic.
// The result of this function should be passed to a subnet manager to update the subnets
// in Azure.
func addEndpointsToSubnets(endpoints []string, subnets []*mgmtnetwork.Subnet) (subnetsToBeUpdated []*mgmtnetwork.Subnet) {
	for _, subnet := range subnets {
		if subnetChanged := addEndpointsToSubnet(endpoints, subnet); subnetChanged {
			subnetsToBeUpdated = append(subnetsToBeUpdated, subnet)
		}
	}

	return subnetsToBeUpdated
}

// addEndpointsToSubnet adds the endpoints (that either are missing in subnet
// or aren't in succeded state in the subnet) to the subnet and returns the updated subnet
func addEndpointsToSubnet(endpoints []string, subnet *mgmtnetwork.Subnet) (subnetChanged bool) {
	for _, endpoint := range endpoints {
		endpointFound, serviceEndpointPtr := subnetContainsEndpoint(subnet, endpoint)

		if !endpointFound || serviceEndpointPtr.ProvisioningState != mgmtnetwork.Succeeded {
			addEndpointToSubnet(endpoint, subnet)
			subnetChanged = true
		}
	}

	return subnetChanged
}

// subnetContainsEndpoint returns false and nil if subnet does not contain the endpoint.
// If the subnet does contain the endpoint, true and a pointer to the service endpoint
// is returned to be able to do additional checks and perform actions accordingly.
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

// addEndpointToSubnet appends the endpoint to the slice of ServiceEndpoints of the subnet.
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
