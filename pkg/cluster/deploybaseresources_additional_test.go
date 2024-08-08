package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
)

func TestDenyAssignment(t *testing.T) {
	m := &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
	}

	tests := []struct {
		Name                      string
		ClusterDocument           *api.OpenShiftClusterDocument
		ExpectedExcludePrincipals *[]mgmtauthorization.Principal
	}{
		{
			Name: "cluster with ServicePrincipalProfile",
			ClusterDocument: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster",
						},
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							SPObjectID: fakeClusterSPObjectId,
						},
					},
				},
			},
			ExpectedExcludePrincipals: &[]mgmtauthorization.Principal{
				{
					ID:   to.StringPtr(fakeClusterSPObjectId),
					Type: to.StringPtr(string(mgmtauthorization.ServicePrincipal)),
				},
			},
		},
		{
			Name: "cluster with PlatformWorkloadIdentityProfile",
			ClusterDocument: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ClusterProfile: api.ClusterProfile{
							ResourceGroupID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster",
						},
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: []api.PlatformWorkloadIdentity{
								{
									OperatorName: "anything",
									ObjectID:     "00000000-0000-0000-0000-000000000000",
									ClientID:     "11111111-1111-1111-1111-111111111111",
									ResourceID:   "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/something/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name",
								},
							},
						},
					},
				},
			},
			ExpectedExcludePrincipals: &[]mgmtauthorization.Principal{
				{
					ID:   to.StringPtr("00000000-0000-0000-0000-000000000000"),
					Type: to.StringPtr(string(mgmtauthorization.ServicePrincipal)),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			m.doc = test.ClusterDocument

			actualDenyAssignment := m.denyAssignment().Resource.(*mgmtauthorization.DenyAssignment)
			actualExcludePrincipals := actualDenyAssignment.ExcludePrincipals

			if !reflect.DeepEqual(test.ExpectedExcludePrincipals, actualExcludePrincipals) {
				t.Errorf("expected %+v, got %+v\n", test.ExpectedExcludePrincipals, actualExcludePrincipals)
			}
		})
	}
}
