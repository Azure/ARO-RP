package permissions

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
)

func TestCanDoAction(t *testing.T) {
	tests := []struct {
		name        string
		permissions []authorization.Permission
		action      string
		want        bool
	}{
		{
			name:   "empty permissions list",
			action: "Microsoft.Network/virtualNetworks/subnets/join/action",
		},
		{
			name: "has permission - exact",
			permissions: []authorization.Permission{
				{
					Actions:    &[]string{"Microsoft.Compute/virtualMachines/*"},
					NotActions: &[]string{},
				},
				{
					Actions:    &[]string{"Microsoft.Network/virtualNetworks/subnets/join/action"},
					NotActions: &[]string{},
				},
			},
			action: "Microsoft.Network/virtualNetworks/subnets/join/action",
			want:   true,
		},
		{
			name: "has permission - wildcard",
			permissions: []authorization.Permission{{
				Actions:    &[]string{"Microsoft.Network/virtualNetworks/subnets/*/action"},
				NotActions: &[]string{},
			}},
			action: "Microsoft.Network/virtualNetworks/subnets/join/action",
			want:   true,
		},
		{
			name: "has permission - exact, conflict",
			permissions: []authorization.Permission{
				{
					Actions:    &[]string{"Microsoft.Network/virtualNetworks/subnets/join/action"},
					NotActions: &[]string{},
				},
				{
					Actions:    &[]string{},
					NotActions: &[]string{"Microsoft.Network/virtualNetworks/subnets/join/action"},
				},
			},
			action: "Microsoft.Network/virtualNetworks/subnets/join/action",
			want:   true,
		},
		{
			name: "has permission excluded - exact",
			permissions: []authorization.Permission{{
				Actions:    &[]string{"Microsoft.Network/*"},
				NotActions: &[]string{"Microsoft.Network/virtualNetworks/subnets/join/action"},
			}},
			action: "Microsoft.Network/virtualNetworks/subnets/join/action",
		},
		{
			name: "has permission excluded - wildcard",
			permissions: []authorization.Permission{{
				Actions:    &[]string{"Microsoft.Network/*"},
				NotActions: &[]string{"Microsoft.Network/virtualNetworks/subnets/*/action"},
			}},
			action: "Microsoft.Network/virtualNetworks/subnets/join/action",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, err := CanDoAction(test.permissions, test.action)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}

			if ok != test.want {
				t.Errorf("expected result %#v, got %#v", test.want, ok)
			}
		})
	}
}
