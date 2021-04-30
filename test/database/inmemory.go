package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

var jsonHandle *codec.JsonHandle

func init() {
	var err error
	jsonHandle, err = database.NewJSONHandle(&fakeAEAD{})
	if err != nil {
		panic(err)
	}
}

func NewFakeOpenShiftClusters() (db database.OpenShiftClusters, client *cosmosdb.FakeOpenShiftClusterDocumentClient) {
	coll := &fakeCollectionClient{}
	client = cosmosdb.NewFakeOpenShiftClusterDocumentClient(jsonHandle)
	injectOpenShiftClusters(client)
	db = database.NewOpenShiftClustersWithProvidedClient(client, coll, "")
	return db, client
}

func NewFakeSubscriptions() (db database.Subscriptions, client *cosmosdb.FakeSubscriptionDocumentClient) {
	client = cosmosdb.NewFakeSubscriptionDocumentClient(jsonHandle)
	injectSubscriptions(client)
	db = database.NewSubscriptionsWithProvidedClient(client, "")
	return db, client
}

func NewFakeBilling() (db database.Billing, client *cosmosdb.FakeBillingDocumentClient) {
	client = cosmosdb.NewFakeBillingDocumentClient(jsonHandle)
	injectBilling(client)
	db = database.NewBillingWithProvidedClient(client)
	return db, client
}

func NewFakeAsyncOperations() (db database.AsyncOperations, client *cosmosdb.FakeAsyncOperationDocumentClient) {
	client = cosmosdb.NewFakeAsyncOperationDocumentClient(jsonHandle)
	db = database.NewAsyncOperationsWithProvidedClient(client)
	return db, client
}

func NewFakePortal() (db database.Portal, client *cosmosdb.FakePortalDocumentClient) {
	client = cosmosdb.NewFakePortalDocumentClient(jsonHandle)
	db = database.NewPortalWithProvidedClient(client)
	return db, client
}

func NewFakeGateway() (db database.Gateway, client *cosmosdb.FakeGatewayDocumentClient) {
	client = cosmosdb.NewFakeGatewayDocumentClient(jsonHandle)
	db = database.NewGatewayWithProvidedClient(client)
	return db, client
}
