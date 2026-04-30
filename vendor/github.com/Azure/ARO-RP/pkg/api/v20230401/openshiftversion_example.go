package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api"

func exampleOpenShiftVersion() *OpenShiftVersion {
	doc := api.ExampleOpenShiftVersionDocument()
	ext := (&openShiftVersionConverter{}).ToExternal(doc.OpenShiftVersion)
	return ext.(*OpenShiftVersion)
}

func ExampleOpenShiftVersionResponse() interface{} {
	return exampleOpenShiftVersion()
}

func ExampleOpenShiftVersionListResponse() interface{} {
	return &OpenShiftVersionList{
		OpenShiftVersions: []*OpenShiftVersion{
			ExampleOpenShiftVersionResponse().(*OpenShiftVersion),
		},
	}
}
