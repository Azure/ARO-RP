package subnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func TestAddEndpointsToSubnets(t *testing.T) {
	var (
		subscriptionId    = "0000000-0000-0000-0000-000000000000"
		vnetResourceGroup = "vnet-rg"
		vnetName          = "vnet"
		subnetNameWorker  = "worker"
		subnetNameMaster  = "master"
		subnetIdWorker    = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameWorker
		subnetIdMaster    = "/subscriptions/" + subscriptionId + "/resourceGroups/" + vnetResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetNameMaster
	)

	type testData struct {
		name            string
		subnets         []*mgmtnetwork.Subnet
		newEndpoints    []string
		expectedSubnets []*mgmtnetwork.Subnet
	}

	tt := []testData{
		{
			name:            "addEndpointsToSubnets should return nil as subnets is nil",
			subnets:         nil,
			newEndpoints:    []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnets: nil,
		},
		{
			name:            "addEndpointsToSubnets should return nil as subnets is an empty slice",
			subnets:         []*mgmtnetwork.Subnet{},
			newEndpoints:    []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnets: nil,
		},
		{
			name: "addEndpointsToSubnets should return nil as all subnets contain all new endpoints and those are in succeeded state",
			subnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
							{
								Service:           to.StringPtr("Microsoft.Storage"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
						},
					},
				},
				{
					ID: to.StringPtr(subnetIdWorker),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
							{
								Service:           to.StringPtr("Microsoft.Storage"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
						},
					},
				},
			},
			newEndpoints:    []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnets: nil,
		},
		{
			name: "addEndpointsToSubnets should return a new updated Subnet because the original subnet's service endpoints is empty",
			subnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{},
					},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:   to.StringPtr("Microsoft.ContainerRegistry"),
								Locations: &[]string{"*"},
							},
							{
								Service:   to.StringPtr("Microsoft.Storage"),
								Locations: &[]string{"*"},
							},
						},
					},
				},
			},
		},
		{
			name: "addEndpointsToSubnets should return a new updated Subnet because the original subnet's service endpoints is nil",
			subnets: []*mgmtnetwork.Subnet{
				{
					ID:                     to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:   to.StringPtr("Microsoft.ContainerRegistry"),
								Locations: &[]string{"*"},
							},
							{
								Service:   to.StringPtr("Microsoft.Storage"),
								Locations: &[]string{"*"},
							},
						},
					},
				},
			},
		},
		{
			name: "addEndpointsToSubnets should return an updated Subnet (with 4 endpoints: 2 previous in failed state + 2 new) as subnet contains all new endpoints but those are not in succeeded state. ",
			subnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Failed,
							},
							{
								Service:           to.StringPtr("Microsoft.Storage"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Failed,
							},
						},
					},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Failed,
							},
							{
								Service:           to.StringPtr("Microsoft.Storage"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Failed,
							},
							{
								Service:   to.StringPtr("Microsoft.ContainerRegistry"),
								Locations: &[]string{"*"},
							},
							{
								Service:   to.StringPtr("Microsoft.Storage"),
								Locations: &[]string{"*"},
							},
						},
					},
				},
			},
		},
		{
			name: "addEndpointsToSubnets should return an updated Subnet (with 2 endpoints: 1 previous was already in succeeded state + 1 new (it was missing))",
			subnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
						},
					},
				},
			},
			newEndpoints: []string{"Microsoft.ContainerRegistry", "Microsoft.Storage"},
			expectedSubnets: []*mgmtnetwork.Subnet{
				{
					ID: to.StringPtr(subnetIdMaster),
					SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
						ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
							{
								Service:           to.StringPtr("Microsoft.ContainerRegistry"),
								Locations:         &[]string{"*"},
								ProvisioningState: mgmtnetwork.Succeeded,
							},
							{
								Service:   to.StringPtr("Microsoft.Storage"),
								Locations: &[]string{"*"},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			subnetsToBeUpdated := addEndpointsToSubnets(tc.newEndpoints, tc.subnets)

			if !reflect.DeepEqual(tc.expectedSubnets, subnetsToBeUpdated) {
				t.Fatalf("expected subnets is different than subnetsToBeUpdated. Expected %v, but got %v", tc.expectedSubnets, subnetsToBeUpdated)
			}
		})
	}
}
