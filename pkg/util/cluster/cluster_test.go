package cluster_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
)

var defaultWIRole api.PlatformWorkloadIdentityRole = api.PlatformWorkloadIdentityRole{
	OperatorName:       "cloud-controller-manager",
	RoleDefinitionName: "Azure Red Hat OpenShift Cloud Controller Manager",
	RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
	ServiceAccounts:    []string{"system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager"},
	SecretLocation: api.SecretLocation{
		Namespace: "openshift-cloud-controller-manager",
		Name:      "azure-cloud-credentials",
	},
}

const oneRoleSet415Raw string = `[
    {
        "openShiftVersion": "4.15",
        "platformWorkloadIdentityRoles": [
            {
                "operatorName": "cloud-controller-manager",
                "roleDefinitionName": "Azure Red Hat OpenShift Cloud Controller Manager",
                "roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
                "serviceAccounts": ["system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager"],
                "secretLocation": { "namespace": "openshift-cloud-controller-manager", "name": "azure-cloud-credentials" }
            }
        ]
}]`

const manyRoleSetsRaw string = `[
{
		"openShiftVersion": "4.16",
		"platformWorkloadIdentityRoles": [
				{
						"operatorName": "cloud-controller-manager-new",
						"roleDefinitionName": "Azure Red Hat OpenShift Cloud Controller Manager",
						"roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
						"serviceAccounts": ["system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager"],
						"secretLocation": { "namespace": "openshift-cloud-controller-manager", "name": "azure-cloud-credentials" }
				}
		]
},
{
		"openShiftVersion": "4.15",
		"platformWorkloadIdentityRoles": [
				{
						"operatorName": "cloud-controller-manager",
						"roleDefinitionName": "Azure Red Hat OpenShift Cloud Controller Manager",
						"roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
						"serviceAccounts": ["system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager"],
						"secretLocation": { "namespace": "openshift-cloud-controller-manager", "name": "azure-cloud-credentials" }
				}
		]
},
{
		"openShiftVersion": "4.14",
		"platformWorkloadIdentityRoles": [
				{
						"operatorName": "cloud-controller-manager-old",
						"roleDefinitionName": "Azure Red Hat OpenShift Cloud Controller Manager",
						"roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/a1f96423-95ce-4224-ab27-4e3dc72facd4",
						"serviceAccounts": ["system:serviceaccount:openshift-cloud-controller-manager:cloud-controller-manager"],
						"secretLocation": { "namespace": "openshift-cloud-controller-manager", "name": "azure-cloud-credentials" }
				}
		]
}
	]`

func TestCluster_GetPlatformWIRoles(t *testing.T) {
	tests := []struct {
		name           string // description of this test case
		roleSetsRaw    string
		clusterVersion string
		want           []api.PlatformWorkloadIdentityRole
		wantErr        bool
	}{
		{
			name:           "No Role sets defined should give an error",
			roleSetsRaw:    "",
			clusterVersion: "4.15.36",
			want:           nil,
			wantErr:        true,
		},
		{
			name:           "Single Roleset with matching version",
			roleSetsRaw:    oneRoleSet415Raw,
			clusterVersion: "4.15.36",
			want:           []api.PlatformWorkloadIdentityRole{defaultWIRole},
			wantErr:        false,
		},
		{
			name:           "Single Roleset with no matching version",
			roleSetsRaw:    oneRoleSet415Raw,
			clusterVersion: "4.14.32",
			want:           nil,
			wantErr:        true,
		},
		{
			name:           "Two Rolesets with one matching version",
			roleSetsRaw:    manyRoleSetsRaw,
			clusterVersion: "4.15.36",
			want:           []api.PlatformWorkloadIdentityRole{defaultWIRole},
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cluster.Cluster{
				Config: &cluster.ClusterConfig{
					WorkloadIdentityRoles: tt.roleSetsRaw,
					OSClusterVersion:      tt.clusterVersion,
				},
			}

			got, gotErr := c.GetPlatformWIRoles()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetPlatformWIRoles() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetPlatformWIRoles() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("GetPlatformWIRoles() = %v, want %v", got, tt.want)
			}
		})
	}
}
