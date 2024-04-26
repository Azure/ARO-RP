package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleSyncIdentityProvider() *SyncIdentityProvider {
	doc := api.ExampleClusterManagerConfigurationDocumentSyncIdentityProvider()
	ext := (&syncIdentityProviderConverter{}).ToExternal(doc.SyncIdentityProvider)
	return ext.(*SyncIdentityProvider)
}

func ExampleSyncIdentityProviderPutParameter() interface{} {
	sip := exampleSyncIdentityProvider()
	sip.ID = ""
	sip.Type = ""
	sip.Name = ""
	return sip
}

func ExampleSyncIdentityProviderPatchParameter() interface{} {
	return ExampleSyncIdentityProviderPutParameter()
}

func ExampleSyncIdentityProviderResponse() interface{} {
	return exampleSyncIdentityProvider()
}

func ExampleSyncIdentityProviderListResponse() interface{} {
	return &SyncIdentityProviderList{
		SyncIdentityProviders: []*SyncIdentityProvider{
			ExampleSyncIdentityProviderResponse().(*SyncIdentityProvider),
		},
	}
}
