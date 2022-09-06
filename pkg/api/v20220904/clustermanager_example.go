package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleSyncSet() *SyncSet {
	doc := api.ExampleClusterManagerConfigurationDocumentSyncSet()
	// ext, err := (&clusterManagerConfigurationConverter{}).ToExternal(doc.ClusterManagerConfiguration)
	ext, err := (&clusterManagerConfigurationConverter{}).SyncSetToExternal(doc.SyncSet)
	if err != nil {
		panic(err)
	}
	return ext.(*SyncSet)
}

func ExampleSyncSetPutParameter() interface{} {
	return exampleSyncSet()
}

func ExampleSyncSetPatchParameter() interface{} {
	return ExampleSyncSetPutParameter()
}

func ExampleSyncSetResponse() interface{} {
	return exampleSyncSet()
}

func ExampleSyncSetListResponse() interface{} {
	return &SyncSetList{
		proxyResource: true,
		SyncSets: []*SyncSet{
			ExampleSyncSetResponse().(*SyncSet),
		},
	}
}
