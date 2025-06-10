package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"

	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestGetDatabaseKey(t *testing.T) {
	primaryMasterKeyName := "PrimaryMasterKey"
	primaryReadOnlyMasterKeyName := "PrimaryReadonlyMasterKey"
	secondaryMasterKeyName := "SecondaryMasterKey"
	secondaryReadOnlyMasterKeyName := "SecondaryReadonlyMasterKey"
	for _, tt := range []struct {
		name        string
		wantData    string
		wantEntries []map[string]types.GomegaMatcher
	}{
		{
			name:     "Use correct CosmosDB Key",
			wantData: secondaryMasterKeyName,
			wantEntries: []map[string]types.GomegaMatcher{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.ContainSubstring(secondaryMasterKeyName),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, log := testlog.New()

			keys := sdkcosmos.DatabaseAccountsClientListKeysResponse{
				DatabaseAccountListKeysResult: sdkcosmos.DatabaseAccountListKeysResult{
					PrimaryMasterKey:           &primaryMasterKeyName,
					PrimaryReadonlyMasterKey:   &primaryReadOnlyMasterKeyName,
					SecondaryMasterKey:         &secondaryMasterKeyName,
					SecondaryReadonlyMasterKey: &secondaryReadOnlyMasterKeyName,
				},
			}

			result := getDatabaseKey(keys, log)
			t.Log(result)

			if result != tt.wantData {
				t.Errorf("Expected %s, got %s", tt.wantData, result)
			}

			err := testlog.AssertLoggingOutput(h, tt.wantEntries)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
