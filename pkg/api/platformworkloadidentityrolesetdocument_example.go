package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ExamplePlatformWorkloadIdentityRoleSetDocument() *PlatformWorkloadIdentityRoleSetDocument {
	return &PlatformWorkloadIdentityRoleSetDocument{
		MissingFields: MissingFields{},
		ID:            "00000000-0000-0000-0000-000000000000",
		PlatformWorkloadIdentityRoleSet: &PlatformWorkloadIdentityRoleSet{
			ID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourceGroupName/providers/resourceProviderNamespace/resourceType/resourceName",
			Name: "4.14",
			Type: "Microsoft.RedHatOpenShift/PlatformWorkloadIdentityRoleSet",
			Properties: PlatformWorkloadIdentityRoleSetProperties{
				OpenShiftVersion: "4.14",
				PlatformWorkloadIdentityRoles: []PlatformWorkloadIdentityRole{
					{
						OperatorName:       "ServiceOperator",
						RoleDefinitionName: "AzureRedHatOpenShiftServiceOperator",
						RoleDefinitionID:   "/providers/Microsoft.Authorization/roleDefinitions/00000000-0000-0000-0000-000000000000",
						ServiceAccounts: []string{
							"aro-operator-master",
						},
					},
				},
			},
		},
	}
}
