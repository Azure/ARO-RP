package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	azcorepolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	sdkcosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	dbmetrics "github.com/Azure/ARO-RP/pkg/metrics/statsd/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcosmos"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

const (
	collAsyncOperations                 = "AsyncOperations"
	collBilling                         = "Billing"
	collClusterManager                  = "ClusterManagerConfigurations"
	collGateway                         = "Gateway"
	collMonitors                        = "Monitors"
	collOpenShiftClusters               = "OpenShiftClusters"
	collOpenShiftVersion                = "OpenShiftVersions"
	collPlatformWorkloadIdentityRoleSet = "PlatformWorkloadIdentityRoleSets"
	collPortal                          = "Portal"
	collSubscriptions                   = "Subscriptions"
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

func NewMasterKeyAuthorizer(ctx context.Context, log *logrus.Entry, token azcore.TokenCredential, clientOptions *policy.ClientOptions, subscriptionID, resourceGroup, databaseAccountName string) (cosmosdb.Authorizer, error) {
	databaseaccounts, err := armcosmos.NewDatabaseAccountsClient(subscriptionID, token, clientOptions)
	if err != nil {
		return nil, err
	}

	// no options defined in the SDK at the moment, but better than passing a nil.
	opt := sdkcosmos.DatabaseAccountsClientListKeysOptions{}
	keys, err := databaseaccounts.ListKeys(ctx, resourceGroup, databaseAccountName, &opt)
	if err != nil {
		return nil, err
	}

	return cosmosdb.NewMasterKeyAuthorizer(getDatabaseKey(keys, log))
}

func NewTokenAuthorizer(ctx context.Context, log *logrus.Entry, cred azcore.TokenCredential, databaseAccountName string, scopes []string) (cosmosdb.Authorizer, error) {
	scopes = append(scopes, fmt.Sprintf("https://%s.documents.azure.com/.default", databaseAccountName))
	acquireToken := func() (token string, newExpiration time.Time, err error) {
		tk, err := cred.GetToken(ctx, azcorepolicy.TokenRequestOptions{Scopes: scopes})
		if err != nil {
			return "", time.Time{}, err
		}
		return tk.Token, tk.ExpiresOn, nil
	}
	tk, expiration, err := acquireToken()
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
