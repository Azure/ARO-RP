package v20240812preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api"

func examplePlatformWorkloadIdentityRoleSet() *PlatformWorkloadIdentityRoleSet {
	doc := api.ExamplePlatformWorkloadIdentityRoleSetDocument()
	ext := (&platformWorkloadIdentityRoleSetConverter{}).ToExternal(doc.PlatformWorkloadIdentityRoleSet)
	return ext.(*PlatformWorkloadIdentityRoleSet)
}

func ExamplePlatformWorkloadIdentityRoleSetResponse() interface{} {
	return examplePlatformWorkloadIdentityRoleSet()
}

func ExamplePlatformWorkloadIdentityRoleSetListResponse() interface{} {
	return &PlatformWorkloadIdentityRoleSetList{
		PlatformWorkloadIdentityRoleSets: []*PlatformWorkloadIdentityRoleSet{
			ExamplePlatformWorkloadIdentityRoleSetResponse().(*PlatformWorkloadIdentityRoleSet),
		},
	}
}
