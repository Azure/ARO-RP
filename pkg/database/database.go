package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"reflect"
	"time"

	azcorepolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	dbmetrics "github.com/Azure/ARO-RP/pkg/metrics/statsd/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

const (
	collAsyncOperations                 = "AsyncOperations"
	collBilling                         = "Billing"
	collGateway                         = "Gateway"
	collMonitors                        = "Monitors"
	collOpenShiftClusters               = "OpenShiftClusters"
	collOpenShiftVersion                = "OpenShiftVersions"
	collPlatformWorkloadIdentityRoleSet = "PlatformWorkloadIdentityRoleSets"
	collPortal                          = "Portal"
	collSubscriptions                   = "Subscriptions"
	collMaintenanceManifests            = "MaintenanceManifests"
)

func NewDatabaseClient(log *logrus.Entry, _env env.Core, authorizer cosmosdb.Authorizer, m metrics.Emitter, aead encryption.AEAD, databaseAccountName string) (cosmosdb.DatabaseClient, error) {
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

	return cosmosdb.NewDatabaseClient(log, c, h, databaseAccountName+"."+_env.Environment().CosmosDBDNSSuffix, authorizer), nil
}

func NewTokenAuthorizer(ctx context.Context, log *logrus.Entry, cred azcore.TokenCredential, databaseAccountName string, scopes []string) (cosmosdb.Authorizer, error) {
	acquireToken := func(contxt context.Context) (token string, newExpiration time.Time, err error) {
		tk, err := cred.GetToken(contxt, azcorepolicy.TokenRequestOptions{Scopes: scopes})
		if err != nil {
			return "", time.Time{}, err
		}
		return tk.Token, tk.ExpiresOn, nil
	}
	tk, expiration, err := acquireToken(ctx)
	if err != nil {
		return nil, err
	}
	return cosmosdb.NewTokenAuthorizer(tk, expiration, acquireToken), nil
}

func getDatabaseKey(keys sdkcosmos.DatabaseAccountsClientListKeysResponse, log *logrus.Entry) string {
	keyName := "SecondaryMasterKey"
	log.Infof("Using %s to authenticate with CosmosDB", keyName)
	return *keys.SecondaryMasterKey
}

func NewJSONHandle(aead encryption.AEAD) (*codec.JsonHandle, error) {
	h := &codec.JsonHandle{}

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
