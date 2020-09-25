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
		name  string
		state string
		want  []string
	}{
		{
			name:  "Not registered for snapshots feature",
			state: "NotRegistered",
			want: []string{
				"Microsoft.Network/networkSecurityGroups/join/action",
			},
		},
		{
			name:  "Registered for snapshots feature",
			state: "Registered",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
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
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  "Microsoft.RedHatOpenShift/EnableSnapshots",
									State: tt.state,
								},
							},
						},
					},
				},
			}
			exceptionsToDeniedActions := *(*((m.denyAssignments("testing").Resource).(*mgmtauthorization.DenyAssignment).
				DenyAssignmentProperties.Permissions))[0].NotActions

			if !reflect.DeepEqual(exceptionsToDeniedActions, tt.want) {
				t.Error(exceptionsToDeniedActions)
			}
		})
	}
}
