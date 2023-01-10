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
	collClusterManager    = "ClusterManagerConfigurations"
	collGateway           = "Gateway"
	collMonitors          = "Monitors"
	collOpenShiftClusters = "OpenShiftClusters"
	collOpenShiftVersion  = "OpenShiftVersions"
	collPortal            = "Portal"
	collSubscriptions     = "Subscriptions"
)

var DefaultClient = &http.Client{}

type MasterKeyClient interface {
	NewMasterKeyAuthorizer(ctx context.Context) (cosmosdb.Authorizer, error)
}

type DatabaseClient interface {
	GetDatabaseClient() (cosmosdb.DatabaseClient, error)
}

type masterKeyClient struct {
	_env                   env.Core
	msiAuthorizer          autorest.Authorizer
	databaseAccountsClient documentdb.DatabaseAccountsClient
}

type databaseClient struct {
	log        *logrus.Entry
	env        env.Core
	authorizer cosmosdb.Authorizer
	httpClient *http.Client
	metrics    metrics.Emitter
	aead       encryption.AEAD
}

func NewDatabaseClient(log *logrus.Entry, env env.Core, authorizer cosmosdb.Authorizer, httpClient *http.Client, metrics metrics.Emitter, aead encryption.AEAD) DatabaseClient {
	return &databaseClient{
		log:        log,
		env:        env,
		authorizer: authorizer,
		httpClient: httpClient,
		metrics:    metrics,
		aead:       aead,
	}
}

func (dc *databaseClient) GetDatabaseClient() (cosmosdb.DatabaseClient, error) {
	for _, key := range []string{
		"DATABASE_ACCOUNT_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	h, err := NewJSONHandle(dc.aead)
	if err != nil {
		return nil, err
	}

	dc.httpClient.Transport = dbmetrics.New(dc.log, &http.Transport{
		// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
		TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
		MaxIdleConnsPerHost: 20,
	}, dc.metrics)
	dc.httpClient.Timeout = 30 * time.Second

	return cosmosdb.NewDatabaseClient(dc.log, dc.httpClient, h, os.Getenv("DATABASE_ACCOUNT_NAME")+"."+dc.env.Environment().CosmosDBDNSSuffix, dc.authorizer), nil
}

func NewMasterKeyClient(_env env.Core, msiAuthorizer autorest.Authorizer, databaseAccountsClient documentdb.NewDatabaseAccountsClient) MasterKeyClient {
	return &masterKeyClient{
		_env:                   _env,
		msiAuthorizer:          msiAuthorizer,
		databaseAccountsClient: databaseAccountsClient.NewDatabaseAccountsClient(_env.Environment(), _env.SubscriptionID(), msiAuthorizer),
	}
}

func (mk *masterKeyClient) NewMasterKeyAuthorizer(ctx context.Context) (cosmosdb.Authorizer, error) {
	for _, key := range []string{
		"DATABASE_ACCOUNT_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	keys, err := mk.databaseAccountsClient.ListKeys(ctx, mk._env.ResourceGroup(), os.Getenv("DATABASE_ACCOUNT_NAME"))
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
