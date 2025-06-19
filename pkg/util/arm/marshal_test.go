package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"testing"

	armcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
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
					Ptr:         pointerutils.ToPtr("test"),
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
					ID:       pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet"),
					Name:     pointerutils.ToPtr("vnet"),
					Type:     pointerutils.ToPtr("microsoft.network/virtualnetworks"),
					Location: pointerutils.ToPtr("eastus"),
					Tags:     map[string]*string{"Tag": pointerutils.ToPtr("Value")},
					Properties: &armnetwork.VirtualNetworkPropertiesFormat{
						ProvisioningState: pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded),
						ResourceGUID:      pointerutils.ToPtr("00000000-0000-0000-0000-000000000000"),
						AddressSpace: &armnetwork.AddressSpace{
							AddressPrefixes: []*string{pointerutils.ToPtr("10.0.0.0/22")},
						},
						Encryption: &armnetwork.VirtualNetworkEncryption{
							Enabled:     pointerutils.ToPtr(false),
							Enforcement: pointerutils.ToPtr(armnetwork.VirtualNetworkEncryptionEnforcementAllowUnencrypted),
						},
						EnableDdosProtection:   pointerutils.ToPtr(false),
						VirtualNetworkPeerings: []*armnetwork.VirtualNetworkPeering{},
						Subnets: []*armnetwork.Subnet{
							{
								ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master"),
								Name: pointerutils.ToPtr("master"),
								Type: pointerutils.ToPtr("Microsoft.Network/virtualNetworks/subnets"),
								Properties: &armnetwork.SubnetPropertiesFormat{
									ProvisioningState:    pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded),
									AddressPrefixes:      []*string{pointerutils.ToPtr("10.0.0.0/23")},
									NetworkSecurityGroup: &armnetwork.SecurityGroup{ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkSecurityGroups/aro-00000-nsg")},
									IPConfigurations: []*armnetwork.IPConfiguration{
										{ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/loadBalancers/ARO-00000-INTERNAL/frontendIPConfigurations/INTERNAL-LB-IP-V4")},
										{ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkInterfaces/ARO-00000-PE.NIC.00000000-0000-0000-0000-000000000000/ipConfigurations/PRIVATEENDPOINTIPCONFIG.00000000-0000-0000-0000-000000000000")},
										{ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkInterfaces/ARO-00000-PLS.NIC.00000000-0000-0000-0000-000000000000/ipConfigurations/ARO-00000-PLS-NIC")},
										{ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/privateLinkServices/ARO-00000-PLS/ipConfigurations/ARO-00000-PLS-NIC")},
									},
									PrivateEndpoints:                  []*armnetwork.PrivateEndpoint{{ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/privateEndpoints/ARO-00000-PE")}},
									PrivateEndpointNetworkPolicies:    pointerutils.ToPtr(armnetwork.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled),
									PrivateLinkServiceNetworkPolicies: pointerutils.ToPtr(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesDisabled),
									Purpose:                           pointerutils.ToPtr("PrivateEndpoints"),
									Delegations:                       []*armnetwork.Delegation{},
								},
							},
							{
								ID:   pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resource-group/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker"),
								Name: pointerutils.ToPtr("worker"),
								Type: pointerutils.ToPtr("Microsoft.Network/virtualNetworks/subnets"),
								Properties: &armnetwork.SubnetPropertiesFormat{
									ProvisioningState:                 pointerutils.ToPtr(armnetwork.ProvisioningStateSucceeded),
									AddressPrefixes:                   []*string{pointerutils.ToPtr("10.0.2.0/23")},
									NetworkSecurityGroup:              &armnetwork.SecurityGroup{ID: pointerutils.ToPtr("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/aro-00000000/providers/Microsoft.Network/networkSecurityGroups/aro-00000-nsg")},
									PrivateEndpointNetworkPolicies:    pointerutils.ToPtr(armnetwork.VirtualNetworkPrivateEndpointNetworkPoliciesDisabled),
									PrivateLinkServiceNetworkPolicies: pointerutils.ToPtr(armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesEnabled),
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
					Type:     pointerutils.ToPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
					Location: pointerutils.ToPtr("eastus"),
					Name:     pointerutils.ToPtr("databaseAccountName/databaseName"),
					Properties: &armcosmos.SQLDatabaseCreateUpdateProperties{
						Options: &armcosmos.CreateUpdateOptions{
							AutoscaleSettings: &armcosmos.AutoscaleSettings{
								MaxThroughput: pointerutils.ToPtr(int32(1000)),
							},
						},
						Resource: &armcosmos.SQLDatabaseResource{
							ID: pointerutils.ToPtr("databaseName"),
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
