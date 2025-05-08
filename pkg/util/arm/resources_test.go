package arm

import (
	"reflect"
	"testing"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func TestArmResources(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *ArmResource
		err   string
	}{
		{
			name:  "happy path split",
			input: "/subscriptions/abc/resourcegroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openshiftclusters/cluster1/syncSets/syncSet1",
			want: &ArmResource{
				SubscriptionID: "abc",
				ResourceGroup:  "v4-eastus",
				Provider:       "Microsoft.RedHatOpenShift",
				ResourceName:   "cluster1",
				ResourceType:   "openshiftclusters",
				SubResource: SubResource{
					ResourceName: "syncSet1",
					ResourceType: "syncSets",
				},
			},
		},
		{
			name:  "happy path - missing subresources",
			input: "/subscriptions/abc/resourcegroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openshiftclusters/cluster1",
			want: &ArmResource{
				SubscriptionID: "abc",
				ResourceGroup:  "v4-eastus",
				Provider:       "Microsoft.RedHatOpenShift",
				ResourceName:   "cluster1",
				ResourceType:   "openshiftclusters",
			},
		},
		{
			name:  "sad path - bad input - missing cluster resource",
			input: "/subscriptions/abc/resourcegroups/v4-eastus/providers",
			err:   "parsing failed for /subscriptions/abc/resourcegroups/v4-eastus/providers. Invalid resource Id format",
		},
		{
			name:  "happy path - two subresources",
			input: "/subscriptions/abc/resourcegroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openshiftclusters/cluster1/syncSets/syncset1/nextResource",
			want: &ArmResource{
				SubscriptionID: "abc",
				ResourceGroup:  "v4-eastus",
				Provider:       "Microsoft.RedHatOpenShift",
				ResourceName:   "cluster1",
				ResourceType:   "openshiftclusters",
				SubResource: SubResource{
					ResourceName: "syncset1",
					ResourceType: "syncSets",
					SubResource: &SubResource{
						ResourceName: "nextResource",
						ResourceType: "syncSets",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ParseArmResourceId(test.input)
			if err != nil {
				if test.err != err.Error() {
					t.Errorf("%s: want %v, got %v", test.name, test.err, err)
				}
			}
			if !reflect.DeepEqual(actual, test.want) {
				t.Errorf("%s: want %v, got %v", test.name, test.want, actual)
			}
		})
	}
}
