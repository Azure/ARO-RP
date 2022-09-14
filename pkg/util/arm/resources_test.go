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
			name:  "sad path - bad input - missing subresources",
			input: "/subscriptions/abc/resourcegroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openshiftclusters/cluster1",
			err:   "parsing failed for /subscriptions/abc/resourcegroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openshiftclusters/cluster1. Invalid resource Id format",
		},
		{
			name:  "sad path - bad input - missing cluster resource",
			input: "/subscriptions/abc/resourcegroups/v4-eastus/providers",
			err:   "parsing failed for /subscriptions/abc/resourcegroups/v4-eastus/providers. Invalid resource Id format",
		},
		{
			name:  "sad path - bad input - too many nested resource",
			input: "/subscriptions/abc/resourcegroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openshiftclusters/cluster1/syncSets/syncset1/nextResource",
			err:   "parsing failed for /subscriptions/abc/resourcegroups/v4-eastus/providers/Microsoft.RedHatOpenShift/openshiftclusters/cluster1/syncSets/syncset1/nextResource. Invalid resource Id format",
		},
	}

	for _, test := range tests {
		actual, err := ParseArmResourceId(test.input)
		if err != nil {
			if test.err != err.Error() {
				t.Fatalf("want %v, got %v", test.err, err)
			}
		}
		if !reflect.DeepEqual(actual, test.want) {
			t.Fatalf("want %v, got %v", test.want, actual)
		}
	}
}
