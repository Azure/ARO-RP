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
	ext, err := (&clusterManagerConfigurationConverter{}).MachinePoolToExternal(doc.MachinePool)
	if err != nil {
		panic(err)
	}
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
	ext, err := (&clusterManagerConfigurationConverter{}).SyncIdentityProviderToExternal(doc.SyncIdentityProvider)
	if err != nil {
		panic(err)
	}
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
	ext, err := (&clusterManagerConfigurationConverter{}).SecretToExternal(doc.Secret)
	if err != nil {
		panic(err)
	}
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
