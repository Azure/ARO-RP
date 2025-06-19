package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	utiljson "github.com/Azure/ARO-RP/test/util/json"
)

func TestResourceMarshal(t *testing.T) {
	tests := []struct {
		name string
		r    *Resource
		want []byte
	}{
		{
			name: "non-zero values",
			r: &Resource{
				Name: "test",
				Resource: &testResource{
					Bool:      true,
					Int:       1,
					Uint:      1,
					Float:     1.1,
					Array:     [1]*testResource{{Bool: true, Unmarshaled: 1}},
					Interface: &testResource{Int: 1, Unmarshaled: 1},
					Map: map[string]*testResource{
						"zero": {Uint: 0, Unmarshaled: 1},
						"one":  {Uint: 1, Unmarshaled: 1},
					},
					Ptr:         to.Ptr("test"),
					Slice:       []*testResource{{Float: 1.1, Unmarshaled: 1}},
					ByteSlice:   []byte("test"),
					String:      "test",
					Struct:      &testResource{String: "test", Unmarshaled: 1},
					Name:        "should be overwritten by parent name",
					Unmarshaled: 1,
					unexported:  1,
				},
			},
			want: []byte(`{
    "bool": true,
    "int": 1,
    "uint": 1,
    "float": 1.1,
    "array": [
        {
            "bool": true,
            "tags": null
        }
    ],
    "interface": {
        "int": 1,
        "tags": null
    },
    "map": {
        "one": {
            "uint": 1,
            "tags": null
        },
        "zero": {
            "tags": null
        }
    },
    "ptr": "test",
    "slice": [
        {
            "float": 1.1,
            "tags": null
        }
    ],
    "byte_slice": "dGVzdA==",
    "string": "test",
    "struct": {
        "string": "test",
        "tags": null
    },
    "name": "test"
}`),
		},
		{
			name: "zero values",
			r: &Resource{
				Name:     "test",
				Resource: &testResource{},
			},
			want: []byte(`{
    "name": "test"
}`),
		},
		{
			name: "vnet",
			r: &Resource{
				APIVersion: "2020-08-01",
				Resource: armnetwork.VirtualNetwork{
					ID:       to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet"),
					Name:     to.Ptr("vnet"),
					Type:     to.Ptr("microsoft.network/virtualnetworks"),
					Location: to.Ptr("eastus"),
					Tags:     map[string]*string{"Tag": to.Ptr("Value")},
					Properties: &armnetwork.VirtualNetworkPropertiesFormat{
						ProvisioningState: to.Ptr(armnetwork.ProvisioningStateSucceeded),
						ResourceGUID:      to.Ptr("00000000-0000-0000-0000-000000000000"),
						AddressSpace: &armnetwork.AddressSpace{
							AddressPrefixes: []*string{to.Ptr("10.0.0.0/22")},
						},
						Encryption: &armnetwork.VirtualNetworkEncryption{
							Enabled:     to.Ptr(false),
							Enforcement: to.Ptr(armnetwork.VirtualNetworkEncryptionEnforcementAllowUnencrypted),
						},
						EnableDdosProtection:   to.Ptr(false),
						VirtualNetworkPeerings: []*armnetwork.VirtualNetworkPeering{},
						Subnets: []*armnetwork.Subnet{
							{
								ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master"),
								Name: to.Ptr("master"),
								Type: to.Ptr("Microsoft.Network/virtualNetworks/subnets"),
								Properties: &armnetwork.SubnetPropertiesFormat{
									ProvisioningState:    to.Ptr(armnetwork.ProvisioningStateSucceeded),
									AddressPrefixes:      []*string{to.Ptr("10.0.0.0/23")},
									NetworkSecurityGroup: &armnetwork.SecurityGroup{ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkSecurityGroups/aro-00000-nsg")},
									IPConfigurations: []*armnetwork.IPConfiguration{
										{ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/loadBalancers/ARO-00000-INTERNAL/frontendIPConfigurations/INTERNAL-LB-IP-V4")},
										{ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkInterfaces/ARO-00000-PE.NIC.00000000-0000-0000-0000-000000000000/ipConfigurations/PRIVATEENDPOINTIPCONFIG.00000000-0000-0000-0000-000000000000")},
										{ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkInterfaces/ARO-00000-PLS.NIC.00000000-0000-0000-0000-000000000000/ipConfigurations/ARO-00000-PLS-NIC")},
										{ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/privateLinkServices/ARO-00000-PLS/ipConfigurations/ARO-00000-PLS-NIC")},
									},
									PrivateEndpoints:                  []*armnetwork.PrivateEndpoint{{ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/privateEndpoints/ARO-00000-PE")}},
									PrivateEndpointNetworkPolicies:    to.Ptr(armnetwork.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled),
									PrivateLinkServiceNetworkPolicies: to.Ptr(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesDisabled),
									Purpose:                           to.Ptr("PrivateEndpoints"),
									Delegations:                       []*armnetwork.Delegation{},
								},
							},
							{
								ID:   to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker"),
								Name: to.Ptr("worker"),
								Type: to.Ptr("Microsoft.Network/virtualNetworks/subnets"),
								Properties: &armnetwork.SubnetPropertiesFormat{
									ProvisioningState:                 to.Ptr(armnetwork.ProvisioningStateSucceeded),
									AddressPrefixes:                   []*string{to.Ptr("10.0.2.0/23")},
									NetworkSecurityGroup:              &armnetwork.SecurityGroup{ID: to.Ptr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkSecurityGroups/aro-00000-nsg")},
									PrivateEndpointNetworkPolicies:    to.Ptr(armnetwork.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled),
									PrivateLinkServiceNetworkPolicies: to.Ptr(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled),
									Delegations:                       []*armnetwork.Delegation{},
								},
							},
						},
					},
				},
			},
			want: []byte(`{
    "apiVersion": "2020-08-01",
    "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet",
    "location": "eastus",
    "name": "vnet",
    "properties": {
        "addressSpace": {
            "addressPrefixes": [
                "10.0.0.0/22"
            ]
        },
        "enableDdosProtection": false,
        "encryption": {
            "enabled": false,
            "enforcement": "AllowUnencrypted"
        },
        "provisioningState": "Succeeded",
        "resourceGuid": "00000000-0000-0000-0000-000000000000",
        "subnets": [
            {
                "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master",
                "name": "master",
                "properties": {
                    "addressPrefixes": [
                        "10.0.0.0/23"
                    ],
                    "networkSecurityGroup": {
                        "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkSecurityGroups/aro-00000-nsg"
                    },
                    "privateEndpointNetworkPolicies": "Disabled",
                    "privateLinkServiceNetworkPolicies": "Disabled",
                    "ipConfigurations": [
                        {
                            "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/loadBalancers/ARO-00000-INTERNAL/frontendIPConfigurations/INTERNAL-LB-IP-V4"
                        },
                        {
                            "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkInterfaces/ARO-00000-PE.NIC.00000000-0000-0000-0000-000000000000/ipConfigurations/PRIVATEENDPOINTIPCONFIG.00000000-0000-0000-0000-000000000000"
                        },
                        {
                            "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkInterfaces/ARO-00000-PLS.NIC.00000000-0000-0000-0000-000000000000/ipConfigurations/ARO-00000-PLS-NIC"
                        },
                        {
                            "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/privateLinkServices/ARO-00000-PLS/ipConfigurations/ARO-00000-PLS-NIC"
                        }
                    ],
                    "privateEndpoints": [
                        {
                            "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/privateEndpoints/ARO-00000-PE"
                        }
                    ],
                    "delegations": [],
                    "provisioningState": "Succeeded",
                    "purpose": "PrivateEndpoints"
                },
                "type": "Microsoft.Network/virtualNetworks/subnets"
            },
            {
                "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker",
                "name": "worker",
                "properties": {
                    "addressPrefixes": [
                        "10.0.2.0/23"
                    ],
                    "networkSecurityGroup": {
                        "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkSecurityGroups/aro-00000-nsg"
                    },
                    "delegations": [],
                    "privateEndpointNetworkPolicies": "Disabled",
                    "privateLinkServiceNetworkPolicies": "Enabled",
                    "provisioningState": "Succeeded"
                },
                "type": "Microsoft.Network/virtualNetworks/subnets"
            }
        ],
        "virtualNetworkPeerings": []
    },
    "tags": {
        "Tag": "Value"
    },
    "type": "microsoft.network/virtualnetworks"
}`),
		},
		{
			name: "database",
			r: &Resource{
				APIVersion: "2023-04-15",
				Resource: &armcosmos.SQLDatabaseCreateUpdateParameters{
					Type:     to.Ptr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
					Location: to.Ptr("eastus"),
					Name:     to.Ptr("databaseAccountName/databaseName"),
					Properties: &armcosmos.SQLDatabaseCreateUpdateProperties{
						Options: &armcosmos.CreateUpdateOptions{
							AutoscaleSettings: &armcosmos.AutoscaleSettings{
								MaxThroughput: to.Ptr(int32(1000)),
							},
						},
						Resource: &armcosmos.SQLDatabaseResource{
							ID: to.Ptr("databaseName"),
						},
					},
				},
			},
			want: []byte(`{
    "apiVersion": "2023-04-15",
    "location": "eastus",
    "name": "databaseAccountName/databaseName",
    "properties": {
        "options": {
            "autoscaleSettings": {
                "maxThroughput": 1000
            }
        },
        "resource": {
            "id": "databaseName"
        }
    },
    "type": "Microsoft.DocumentDB/databaseAccounts/sqlDatabases"
}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := json.MarshalIndent(test.r, "", "    ")
			if err != nil {
				t.Fatal(err)
			}

			utiljson.AssertJsonMatches(t, test.want, b)
		})
	}
}

type testResource struct {
	Bool        bool                     `json:"bool,omitempty"`
	Int         int                      `json:"int,omitempty"`
	Uint        uint                     `json:"uint,omitempty"`
	Float       float64                  `json:"float,omitempty"`
	Array       [1]*testResource         `json:"array,omitempty"`
	Interface   interface{}              `json:"interface,omitempty"`
	Map         map[string]*testResource `json:"map,omitempty"`
	Ptr         *string                  `json:"ptr,omitempty"`
	Slice       []*testResource          `json:"slice,omitempty"`
	ByteSlice   []byte                   `json:"byte_slice,omitempty"`
	String      string                   `json:"string,omitempty"`
	Struct      *testResource            `json:"struct,omitempty"`
	Name        string                   `json:"name,omitempty"`
	Unmarshaled int                      `json:"-"`
	unexported  int
	// Both `arm.Resource` and nested `testResource` have fields with name `Tags`.
	// The `Tags` field from `arm.Resource` must override the one from `testResource`
	// on the top-level of JSON.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON contains custom marshaling logic which we expect to be dropped
// during marshalling as part of arm.Resource type
func (r *testResource) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("should not be called")
}
