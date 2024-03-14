package keysprovider

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// package keyprovider ensures we are accessing the correct keys by forcing that at compile time
// but just providing the public methods we want.

import (
	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
)

type ContextInfo string

type KeyInfo struct {
	//The actual raw value of the key
	Value string

	// Contextual information about the key
	ContextInfo ContextInfo
}

// DatabaseKeysProvider is responsible to ensure we use the appropiate keys from CosmosDB.
// Instead of manipulating directly the keys we get, we use this level of
// indirection and we just provide a public method to use the only key we want to ensure we use.
type DatabaseKeysProvider struct {
	// keys is an unexported field, don't want to give direct access from outside
	keys sdkcosmos.DatabaseAccountsClientListKeysResponse
}

func NewDatabaseKeysProvider(keys sdkcosmos.DatabaseAccountsClientListKeysResponse) DatabaseKeysProvider {
	return DatabaseKeysProvider{
		keys: keys,
	}
}

// GetSecondaryMasterKey ensures we use the SecondaryMasterKey from Azure CosmosDB keys
func (provider DatabaseKeysProvider) GetSecondaryMasterKey() KeyInfo {
	return KeyInfo{
		Value:       *provider.keys.SecondaryMasterKey,
		ContextInfo: "Using SecondaryMasterKey to authenticate with CosmosDB",
	}
}

// We could implement a different method depending on the case.
// This is left for informative purposes. To be deleted. :)

// func (provider DatabaseKeysProvider) GetPrimaryMasterKey() KeyInfo {
// 	return KeyInfo{
// 		keyValue: *provider.keys.PrimaryMasterKey,
// 		keyInfo:  "Using PrimaryMasterKey to authenticate with CosmosDB",
// 	}
// }
// etc
