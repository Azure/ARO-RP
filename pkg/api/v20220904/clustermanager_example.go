package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

func exampleSyncSet() *SyncSet {
	doc := api.ExampleClusterManagerConfigurationDocumentSyncSet()
	ext := (&clusterManagerConfigurationConverter{}).SyncSetToExternal(doc.SyncSet)
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

func exampleMachinePool() *MachinePool {
	doc := api.ExampleClusterManagerConfigurationDocumentMachinePool()
	ext := (&clusterManagerConfigurationConverter{}).MachinePoolToExternal(doc.MachinePool)
	return ext.(*MachinePool)
}

func ExampleMachinePoolPutParameter() interface{} {
	mp := exampleMachinePool()
	mp.ID = ""
	mp.Type = ""
	mp.Name = ""
	return mp
}

func ExampleMachinePoolPatchParameter() interface{} {
	return ExampleMachinePoolPutParameter()
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
	ext := (&clusterManagerConfigurationConverter{}).SyncIdentityProviderToExternal(doc.SyncIdentityProvider)
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

func exampleSecret() *Secret {
	doc := api.ExampleClusterManagerConfigurationDocumentSecret()
	ext := (&clusterManagerConfigurationConverter{}).SecretToExternal(doc.Secret)
	return ext.(*Secret)
}

func ExampleSecretPutParameter() interface{} {
	s := exampleSecret()
	s.ID = ""
	s.Type = ""
	s.Name = ""
	return s
}

func ExampleSecretPatchParameter() interface{} {
	return ExampleSecretPutParameter()
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
