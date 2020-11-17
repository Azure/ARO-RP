package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	dbmetrics "github.com/Azure/ARO-RP/pkg/metrics/statsd/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/documentdb"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

const (
	collAsyncOperations   = "AsyncOperations"
	collBilling           = "Billing"
	collMonitors          = "Monitors"
	collOpenShiftClusters = "OpenShiftClusters"
	collSubscriptions     = "Subscriptions"
)

func NewDatabaseClient(ctx context.Context, log *logrus.Entry, env env.Core, m metrics.Interface, cipher encryption.Cipher) (cosmosdb.DatabaseClient, error) {
	databaseAccount, masterKey, err := find(ctx, env)
	if err != nil {
		return nil, err
	}

	h := NewJSONHandle(cipher)

	c := &http.Client{
		Transport: dbmetrics.New(log, &http.Transport{
			// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
			TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
			MaxIdleConnsPerHost: 20,
		}, m),
		Timeout: 30 * time.Second,
	}

	databaseHostname := databaseAccount + "." + env.Environment().CosmosDBDNSSuffix
	return cosmosdb.NewDatabaseClient(log, c, h, databaseHostname, masterKey)
}

func NewJSONHandle(cipher encryption.Cipher) *codec.JsonHandle {
	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
	}

	h.SetInterfaceExt(reflect.TypeOf(api.SecureBytes{}), 1, secureBytesExt{cipher: cipher})
	h.SetInterfaceExt(reflect.TypeOf((*api.SecureString)(nil)), 1, secureStringExt{cipher: cipher})
	return h
}

func databaseName(deploymentMode deployment.Mode) (string, error) {
	if deploymentMode != deployment.Development {
		return "ARO", nil
	}

	for _, key := range []string{
		"DATABASE_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return "", fmt.Errorf("environment variable %q unset (development mode)", key)
		}
	}

	return os.Getenv("DATABASE_NAME"), nil
}

func find(ctx context.Context, env env.Core) (string, string, error) {
	rpAuthorizer, err := env.NewRPAuthorizer(env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return "", "", err
	}

	databaseaccounts := documentdb.NewDatabaseAccountsClient(env.SubscriptionID(), rpAuthorizer)

	acctName := os.Getenv("DATABASE_ACCOUNT_NAME")
	if acctName == "" {
		accts, err := databaseaccounts.ListByResourceGroup(ctx, env.ResourceGroup())
		if err != nil {
			return "", "", err
		}

		if len(*accts.Value) != 1 {
			return "", "", fmt.Errorf("found %d database accounts, expected 1", len(*accts.Value))
		}

		acctName = *(*accts.Value)[0].Name
	}

	keys, err := databaseaccounts.ListKeys(ctx, env.ResourceGroup(), acctName)
	if err != nil {
		return "", "", err
	}

	return acctName, *keys.PrimaryMasterKey, nil
}
