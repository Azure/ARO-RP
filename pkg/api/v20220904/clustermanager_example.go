package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleSyncSet() *SyncSet {
	doc := api.ExampleClusterManagerConfigurationDocumentSyncSet()
	doc.SyncSet.ID = ""
	doc.SyncSet.Type = ""
	doc.SyncSet.Name = ""
	ext := (&clusterManagerConfigurationConverter{}).SyncSetToExternal(doc.SyncSet)
	return ext.(*SyncSet)
}

func ExampleSyncSetPutParameter() interface{} {
	return exampleSyncSet()
}

func ExampleSyncSetPatchParameter() interface{} {
	return exampleSyncSet()
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

func exampleMachinePool() *MachinePool {
	doc := api.ExampleClusterManagerConfigurationDocumentMachinePool()
	doc.MachinePool.ID = ""
	doc.MachinePool.Type = ""
	doc.MachinePool.Name = ""
	ext := (&clusterManagerConfigurationConverter{}).MachinePoolToExternal(doc.MachinePool)

	return ext.(*MachinePool)
}

func ExampleMachinePoolPutParameter() interface{} {
	return exampleMachinePool()
}

func ExampleMachinePoolPatchParameter() interface{} {
	return exampleMachinePool()
}

func ExampleMachinePoolResponse() interface{} {
	return exampleMachinePool()
}

func ExampleMachinePoolListResponse() interface{} {
	return &MachinePoolList{
		MachinePools: []*MachinePool{
			ExampleMachinePoolResponse().(*MachinePool),
		},
	}
}

func exampleSyncIdentityProvider() *SyncIdentityProvider {
	doc := api.ExampleClusterManagerConfigurationDocumentSyncIdentityProvider()
	doc.SyncIdentityProvider.ID = ""
	doc.SyncIdentityProvider.Type = ""
	doc.SyncIdentityProvider.Name = ""
	ext := (&clusterManagerConfigurationConverter{}).SyncIdentityProviderToExternal(doc.SyncIdentityProvider)
	return ext.(*SyncIdentityProvider)
}

func ExampleSyncIdentityProviderPutParameter() interface{} {
	return exampleSyncIdentityProvider()
}

func ExampleSyncIdentityProviderPatchParameter() interface{} {
	return exampleSyncIdentityProvider()
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

func exampleSecret() *Secret {
	doc := api.ExampleClusterManagerConfigurationDocumentSecret()
	doc.Secret.ID = ""
	doc.Secret.Type = ""
	doc.Secret.Name = ""
	ext := (&clusterManagerConfigurationConverter{}).SecretToExternal(doc.Secret)
	return ext.(*Secret)
}

func ExampleSecretPutParameter() interface{} {
	return exampleSecret()
}

func ExampleSecretPatchParameter() interface{} {
	return exampleSecret()
}

func ExampleSecretResponse() interface{} {
	return exampleSecret()
}

func ExampleSecretListResponse() interface{} {
	return &SecretList{
		Secrets: []*Secret{
			ExampleSecretResponse().(*Secret),
		},
	}
}
