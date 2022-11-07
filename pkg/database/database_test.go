package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	mgmtdocumentdb "github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2021-01-15/documentdb"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_documentdb "github.com/Azure/ARO-RP/pkg/util/mocks/documentdb"
)

var (
	databaseAccount = "testdb"
	envVars         = map[string]string{
		auth.SubscriptionID:      "my-subscription",
		auth.TenantID:            "my-tenant",
		auth.AuxiliaryTenantIDs:  "aux-tenants",
		auth.ClientID:            "my-clientid",
		auth.ClientSecret:        "my-client-secret",
		auth.CertificatePath:     "my-cert-path",
		auth.CertificatePassword: "my-cert-pass",
		auth.Username:            "my-username",
		auth.Password:            "my-password",
		"RESOURCEGROUP":          ".my-resource.test.com",
		auth.Resource:            "azure-resource",
		"DATABASE_ACCOUNT_NAME":  "testdb",
		"DATABASE_NAME":          "testdb",
		"RP_MODE":                "development",
		"LOCATION":               "local",
		"KEYVAULT_PREFIX":        "my-keyvault",
		"AZURE_RP_CLIENT_ID":     "azure-rp",
		"AZURE_RP_CLIENT_SECRET": "azure-rp-secret",
	}
)

func TestGetDatabaseClient(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		wantErr string
		unset   string
		envCore *env.Core
	}{
		{
			name:    "fail: DATABASE_ACCOUNT_NAME unset",
			wantErr: "environment variable \"DATABASE_ACCOUNT_NAME\" unset",
			unset:   "DATABASE_ACCOUNT_NAME",
		},
		{
			name: "pass: Create new database client",
		},
	} {
		for v, k := range envVars {
			os.Setenv(v, k)
		}
		if tt.unset != "" {
			os.Unsetenv(tt.unset)
		}

		_env, err := env.NewCore(ctx, log)
		if err != nil {
			t.Errorf("failed to get env core, %v", err)
		}

		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "dbs/testdb/colls/Subscriptions/triggers" {
				w.WriteHeader(http.StatusOK)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		r := rand.New(rand.NewSource(time.Now().UnixMicro()))
		authorizer := cosmosdb.NewTokenAuthorizer(fmt.Sprintf("rand-%d", r.Int()))

		dc := NewDatabaseClient(log, _env, authorizer, ts.Client(), &noop.Noop{}, nil)

		t.Run(tt.name, func(t *testing.T) {
			_, err := dc.GetDatabaseClient()
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestName(t *testing.T) {
	for _, tt := range []struct {
		name    string
		wantErr string
		unset   string
		devMode bool
	}{
		{
			name:    "fail: unset DATABASE_NAME",
			wantErr: "environment variable \"DATABASE_NAME\" unset (development mode)",
			devMode: true,
			unset:   "DATABASE_NAME",
		},
		{
			name:    "pass: not development environment",
			devMode: false,
		},
		{
			name:    "pass: get DATABASE_NAME",
			devMode: true,
		},
	} {
		for v, k := range envVars {
			os.Setenv(v, k)
		}
		if tt.unset != "" {
			os.Unsetenv(tt.unset)
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := Name(tt.devMode)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestNewMasterKeyAuthorizer(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())
	ctx := context.Background()

	for _, tt := range []struct {
		name             string
		wantErr          string
		mockErr          error
		databaseAccount  string
		keys             mgmtdocumentdb.DatabaseAccountListKeysResult
		primaryMasterKey string
		unset            string
	}{
		{
			name:             "fail: empty DATABASE_ACCOUNT_NAME",
			primaryMasterKey: base64.StdEncoding.EncodeToString([]byte("primary-master-key")),
			wantErr:          "environment variable \"DATABASE_ACCOUNT_NAME\" unset",
			unset:            "DATABASE_ACCOUNT_NAME",
		},
		{
			name:    "fail: database accounts list keys",
			mockErr: fmt.Errorf("documentdb.DatabaseAccountsClient#ListKeys: Invalid input: autorest/validation: validation failed: parameter=client.SubscriptionID constraint=MinLength value=\"\" details: value length must be greater than or equal to 1"),
			wantErr: "documentdb.DatabaseAccountsClient#ListKeys: Invalid input: autorest/validation: validation failed: parameter=client.SubscriptionID constraint=MinLength value=\"\" details: value length must be greater than or equal to 1",
		},
		{
			name:             "pass: successfully create master key authorizer",
			primaryMasterKey: base64.StdEncoding.EncodeToString([]byte("primary-master-key")),
		},
	} {
		for env, val := range envVars {
			os.Setenv(env, val)
		}
		if tt.unset != "" {
			os.Unsetenv(tt.unset)
		}

		keys := mgmtdocumentdb.DatabaseAccountListKeysResult{
			PrimaryMasterKey: &tt.primaryMasterKey,
		}

		_env, err := env.NewCore(ctx, log)
		if err != nil {
			t.Errorf("failed to create envCore, %v", err)
		}

		msiAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
		if err != nil {
			t.Error(err)
		}

		controller := gomock.NewController(t)
		documentdb := mock_documentdb.NewMockNewDatabaseAccountsClient(controller)

		documentdb.EXPECT().NewDatabaseAccountsClient(_env.Environment(), _env.SubscriptionID(), msiAuthorizer).MaxTimes(1).Return(documentdb)
		masterKeyClient := NewMasterKeyClient(_env, msiAuthorizer, documentdb)
		documentdb.EXPECT().ListKeys(ctx, _env.ResourceGroup(), databaseAccount).MaxTimes(1).Return(keys, tt.mockErr)

		t.Run(tt.name, func(t *testing.T) {
			_, err = masterKeyClient.NewMasterKeyAuthorizer(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
		for env := range envVars {
			os.Unsetenv(env)
		}
	}
}
