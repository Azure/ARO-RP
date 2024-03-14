package keysprovider

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
)

func TestDatabaseKeyProvider(t *testing.T) {
	primaryMasterKeyName := "PrimaryMasterKey"
	primaryReadOnlyMasterKeyName := "PrimaryReadonlyMasterKey"
	secondaryMasterKeyName := "SecondaryMasterKey"
	secondaryReadOnlyMasterKeyName := "SecondaryReadonlyMasterKey"

	for _, tt := range []struct {
		name        string
		wantKeyInfo KeyInfo
	}{
		{
			name: "Use correct CosmosDB Key",
			wantKeyInfo: KeyInfo{
				Value:       secondaryMasterKeyName,
				ContextInfo: "Using SecondaryMasterKey to authenticate with CosmosDB",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			keys := sdkcosmos.DatabaseAccountsClientListKeysResponse{
				DatabaseAccountListKeysResult: sdkcosmos.DatabaseAccountListKeysResult{
					PrimaryMasterKey:           &primaryMasterKeyName,
					PrimaryReadonlyMasterKey:   &primaryReadOnlyMasterKeyName,
					SecondaryMasterKey:         &secondaryMasterKeyName,
					SecondaryReadonlyMasterKey: &secondaryReadOnlyMasterKeyName,
				},
			}

			keysProvider := NewDatabaseKeysProvider(keys)
			keyInfo := keysProvider.GetSecondaryMasterKey()

			if !reflect.DeepEqual(tt.wantKeyInfo, keyInfo) {
				t.Errorf("Want %+v", tt.wantKeyInfo)
				t.Errorf("But got %+v", keyInfo)
			}
		})
	}
}
