package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleClusterManagerConfigurationSyncSet() *ClusterManagerConfiguration {
	doc := api.ExampleClusterManagerConfigurationDocumentSyncSet()
	ext, err := (&clusterManagerConverter{}).ToExternal(doc.ClusterManagerConfiguration)
	if err != nil {
		panic(err)
	}
	return ext.(*ClusterManagerConfiguration)
}

func ExampleClusterManagerSyncSetPutParameter() interface{} {
	ocm := exampleClusterManagerConfigurationSyncSet()
	return ocm
}

func ExampleClusterManagerSyncSetPatchParameter() interface{} {
	ocm := ExampleClusterManagerSyncSetPutParameter()
	return ocm
}

func ExampleClusterManagerSyncSetResponse() interface{} {
	ocm := exampleClusterManagerConfigurationSyncSet()
	return ocm
}

func ExampleClusterManagerListSyncSetResponse() interface{} {
	return &ClusterManagerConfigurationsList{
		ClusterManagerConfigurations: []*ClusterManagerConfiguration{
			ExampleClusterManagerSyncSetResponse().(*ClusterManagerConfiguration),
		},
	}
}
