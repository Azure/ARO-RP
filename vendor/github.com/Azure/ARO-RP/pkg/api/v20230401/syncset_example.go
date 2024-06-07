package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleSyncSet() *SyncSet {
	doc := api.ExampleClusterManagerConfigurationDocumentSyncSet()
	ext := (&syncSetConverter{}).ToExternal(doc.SyncSet)
	return ext.(*SyncSet)
}

func ExampleSyncSetPutParameter() interface{} {
	ss := exampleSyncSet()
	ss.ID = ""
	ss.Type = ""
	ss.Name = ""
	return ss
}

func ExampleSyncSetPatchParameter() interface{} {
	return ExampleSyncSetPutParameter()
}

func ExampleSyncSetResponse() interface{} {
	return exampleSyncSet()
}

func ExampleSyncSetListResponse() interface{} {
	return &SyncSetList{
		SyncSets: []*SyncSet{
			ExampleSyncSetResponse().(*SyncSet),
		},
	}
}
