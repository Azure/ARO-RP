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

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	dbmetrics "github.com/Azure/ARO-RP/pkg/metrics/statsd/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/documentdb"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

const (
	collAsyncOperations   = "AsyncOperations"
	collBilling           = "Billing"
	collClusterManager    = "ClusterManager"
	collGateway           = "Gateway"
	collHiveResources     = "HiveResources"
	collMonitors          = "Monitors"
	collOpenShiftClusters = "OpenShiftClusters"
	collPortal            = "Portal"
	collSubscriptions     = "Subscriptions"
	collOpenShiftVersion  = "OpenShiftVersions"
)

func NewDatabaseClient(log *logrus.Entry, env env.Core, authorizer cosmosdb.Authorizer, m metrics.Emitter, aead encryption.AEAD) (cosmosdb.DatabaseClient, error) {
	for _, key := range []string{
		"DATABASE_ACCOUNT_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	h, err := NewJSONHandle(aead)
	if err != nil {
		return nil, err
	}

	c := &http.Client{
		Transport: dbmetrics.New(log, &http.Transport{
			// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
			TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
			MaxIdleConnsPerHost: 20,
		}, m),
		Timeout: 30 * time.Second,
	}

	return cosmosdb.NewDatabaseClient(log, c, h, os.Getenv("DATABASE_ACCOUNT_NAME")+"."+env.Environment().CosmosDBDNSSuffix, authorizer), nil
}

func NewMasterKeyAuthorizer(ctx context.Context, _env env.Core, msiAuthorizer autorest.Authorizer) (cosmosdb.Authorizer, error) {
	for _, key := range []string{
		"DATABASE_ACCOUNT_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	databaseaccounts := documentdb.NewDatabaseAccountsClient(_env.Environment(), _env.SubscriptionID(), msiAuthorizer)

	keys, err := databaseaccounts.ListKeys(ctx, _env.ResourceGroup(), os.Getenv("DATABASE_ACCOUNT_NAME"))
	if err != nil {
		return nil, err
	}

	return cosmosdb.NewMasterKeyAuthorizer(*keys.PrimaryMasterKey)
}

func NewJSONHandle(aead encryption.AEAD) (*codec.JsonHandle, error) {
	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
	}

	if aead == nil {
		return h, nil
	}

	err := h.SetInterfaceExt(reflect.TypeOf(api.SecureBytes{}), 1, secureBytesExt{aead: aead})
	if err != nil {
		return nil, err
	}

	err = h.SetInterfaceExt(reflect.TypeOf((*api.SecureString)(nil)), 1, secureStringExt{aead: aead})
	if err != nil {
		return nil, err
	}

	return h, nil
}

func Name(isLocalDevelopmentMode bool) (string, error) {
	if !isLocalDevelopmentMode {
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
