package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	dbmetrics "github.com/Azure/ARO-RP/pkg/metrics/statsd/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func NewDatabaseClient(ctx context.Context, log *logrus.Entry, env env.Interface, m metrics.Interface, cipher encryption.Cipher) (cosmosdb.DatabaseClient, error) {
	databaseAccount, masterKey := env.CosmosDB()

	h := newJSONHandle(cipher)

	c := &http.Client{
		Transport: dbmetrics.New(log, &http.Transport{
			// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
			TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
			MaxIdleConnsPerHost: 20,
		}, m),
		Timeout: 30 * time.Second,
	}

	return cosmosdb.NewDatabaseClient(log, c, h, databaseAccount, masterKey)
}

/*
	db.AsyncOperations, err = NewAsyncOperations(env, dbc)
	if err != nil {
		return nil, err
	}

	db.Billing, err = NewBilling(ctx, env, dbc)
	if err != nil {
		return nil, err
	}

	db.Monitors, err = NewMonitors(ctx, env, dbc, uuid)
	if err != nil {
		return nil, err
	}

	db.OpenShiftClusters, err = NewOpenShiftClusters(ctx, env, dbc, uuid)
	if err != nil {
		return nil, err
	}

	db.Subscriptions, err = NewSubscriptions(ctx, env, dbc, uuid)
	if err != nil {
		return nil, err
	}
*/

func newJSONHandle(cipher encryption.Cipher) *codec.JsonHandle {
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
