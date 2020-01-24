package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
)

// Database represents a database
type Database struct {
	AsyncOperations   AsyncOperations
	Monitors          Monitors
	OpenShiftClusters OpenShiftClusters
	Subscriptions     Subscriptions
}

// NewDatabase returns a new Database
func NewDatabase(ctx context.Context, log *logrus.Entry, env env.Interface, uuid string) (db *Database, err error) {
	databaseAccount, masterKey := env.CosmosDB()

	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
	}

	err = api.AddExtensions(&h.BasicHandle)
	if err != nil {
		return nil, err
	}

	c := &http.Client{
		Transport: &http.Transport{
			// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
			TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
		},
		Timeout: 30 * time.Second,
	}

	dbc, err := cosmosdb.NewDatabaseClient(log, c, h, databaseAccount, masterKey)
	if err != nil {
		return nil, err
	}

	db = &Database{}

	db.AsyncOperations, err = NewAsyncOperations(uuid, dbc, env.DatabaseName(), "AsyncOperations")
	if err != nil {
		return nil, err
	}

	db.Monitors, err = NewMonitors(ctx, uuid, dbc, env.DatabaseName(), "Monitors")
	if err != nil {
		return nil, err
	}

	db.OpenShiftClusters, err = NewOpenShiftClusters(ctx, uuid, dbc, env.DatabaseName(), "OpenShiftClusters")
	if err != nil {
		return nil, err
	}

	db.Subscriptions, err = NewSubscriptions(ctx, uuid, dbc, env.DatabaseName(), "Subscriptions")
	if err != nil {
		return nil, err
	}

	return db, nil
}
