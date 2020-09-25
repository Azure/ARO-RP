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

	"github.com/Azure/go-autorest/autorest/azure"
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

// Database represents a database
type Database struct {
	AsyncOperations   AsyncOperations
	Billing           Billing
	Monitors          Monitors
	OpenShiftClusters OpenShiftClusters
	Subscriptions     Subscriptions
}

// NewDatabase returns a new Database
func NewDatabase(ctx context.Context, log *logrus.Entry, env env.Core, m metrics.Interface, cipher encryption.Cipher, uuid string) (db *Database, err error) {
	databaseName, err := databaseName(env.DeploymentMode())
	if err != nil {
		return nil, err
	}

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

	dbc, err := cosmosdb.NewDatabaseClient(log, c, h, databaseAccount, masterKey)
	if err != nil {
		return nil, err
	}

	db = &Database{}

	db.AsyncOperations, err = NewAsyncOperations(uuid, dbc, databaseName, "AsyncOperations")
	if err != nil {
		return nil, err
	}

	db.Billing, err = NewBilling(ctx, uuid, dbc, databaseName, "Billing")
	if err != nil {
		return nil, err
	}

	db.Monitors, err = NewMonitors(ctx, uuid, dbc, databaseName, "Monitors")
	if err != nil {
		return nil, err
	}

	db.OpenShiftClusters, err = NewOpenShiftClusters(ctx, uuid, dbc, databaseName, "OpenShiftClusters")
	if err != nil {
		return nil, err
	}

	db.Subscriptions, err = NewSubscriptions(ctx, uuid, dbc, databaseName, "Subscriptions")
	if err != nil {
		return nil, err
	}

	return db, nil
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
	rpAuthorizer, err := env.NewRPAuthorizer(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", "", err
	}

	databaseaccounts := documentdb.NewDatabaseAccountsClient(env.SubscriptionID(), rpAuthorizer)

	accts, err := databaseaccounts.ListByResourceGroup(ctx, env.ResourceGroup())
	if err != nil {
		return "", "", err
	}

	if len(*accts.Value) != 1 {
		return "", "", fmt.Errorf("found %d database accounts, expected 1", len(*accts.Value))
	}

	keys, err := databaseaccounts.ListKeys(ctx, env.ResourceGroup(), *(*accts.Value)[0].Name)
	if err != nil {
		return "", "", err
	}

	return *(*accts.Value)[0].Name, *keys.PrimaryMasterKey, nil
}
