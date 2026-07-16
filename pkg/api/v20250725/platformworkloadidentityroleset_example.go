package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api"

func examplePlatformWorkloadIdentityRoleSet() *PlatformWorkloadIdentityRoleSet {
	doc := api.ExamplePlatformWorkloadIdentityRoleSetDocument()
	ext := (&platformWorkloadIdentityRoleSetConverter{}).ToExternal(doc.PlatformWorkloadIdentityRoleSet)
	return ext.(*PlatformWorkloadIdentityRoleSet)
}

func ExamplePlatformWorkloadIdentityRoleSetResponse() any {
	return examplePlatformWorkloadIdentityRoleSet()
}

func ExamplePlatformWorkloadIdentityRoleSetListResponse() any {
	return &PlatformWorkloadIdentityRoleSetList{
		PlatformWorkloadIdentityRoleSets: []*PlatformWorkloadIdentityRoleSet{
			ExamplePlatformWorkloadIdentityRoleSetResponse().(*PlatformWorkloadIdentityRoleSet),
		},
	}
}
