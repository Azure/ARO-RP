package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestDenyAssignments(t *testing.T) {
	for _, tt := range []struct {
		name         string
		featureFlags []string
		want         []string
	}{
		{
			name: "Not registered for snapshots feature",
			want: []string{
				"Microsoft.Network/networkSecurityGroups/join/action",
				"Microsoft.Compute/disks/beginGetAccess/action",
				"Microsoft.Compute/disks/endGetAccess/action",
				"Microsoft.Compute/disks/write",
				"Microsoft.Compute/snapshots/beginGetAccess/action",
				"Microsoft.Compute/snapshots/endGetAccess/action",
				"Microsoft.Compute/snapshots/write",
				"Microsoft.Compute/snapshots/delete",
			},
		},
		{
			name:         "Registered for engineering feature flag",
			featureFlags: []string{"Microsoft.RedHatOpenShift/RedHatEngineering"},
			want: []string{
				"Microsoft.Network/networkSecurityGroups/join/action",
				"Microsoft.Compute/disks/beginGetAccess/action",
				"Microsoft.Compute/disks/endGetAccess/action",
				"Microsoft.Compute/disks/write",
				"Microsoft.Compute/snapshots/beginGetAccess/action",
				"Microsoft.Compute/snapshots/endGetAccess/action",
				"Microsoft.Compute/snapshots/write",
				"Microsoft.Compute/snapshots/delete",
				"Microsoft.Network/networkInterfaces/effectiveRouteTable/action",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var features = []api.RegisteredFeatureProfile{}
			for i := range tt.featureFlags {
				features = append(features, api.RegisteredFeatureProfile{
					Name:  tt.featureFlags[i],
					State: "Registered",
				})
			}
			m := &manager{
				doc: &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: "testing",
							},
						},
					},
				},
				subscriptionDoc: &api.SubscriptionDocument{
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							RegisteredFeatures: features,
						},
					},
				},
			}
			exceptionsToDeniedActions := *(*((m.denyAssignments().Resource).(*mgmtauthorization.DenyAssignment).
				DenyAssignmentProperties.Permissions))[0].NotActions
			if !reflect.DeepEqual(exceptionsToDeniedActions, tt.want) {
				t.Error(exceptionsToDeniedActions)
			}
		})
	}
}
