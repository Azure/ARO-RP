package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

var jsonHandle *codec.JsonHandle = database.NewJSONHandle(&fakeCipher{})

func NewFakeOpenShiftClusters() (db database.OpenShiftClusters, client *cosmosdb.FakeOpenShiftClusterDocumentClient, _uuid string) {
	_uuid = uuid.NewV4().String()
	coll := &fakeCollectionClient{}
	client = cosmosdb.NewFakeOpenShiftClusterDocumentClient(jsonHandle)
	injectOpenShiftClusters(client)
	db = database.NewOpenShiftClustersWithProvidedClient(_uuid, client, coll)
	return db, client, _uuid
}

func NewFakeSubscriptions() (db database.Subscriptions, client *cosmosdb.FakeSubscriptionDocumentClient, _uuid string) {
	_uuid = uuid.NewV4().String()
	client = cosmosdb.NewFakeSubscriptionDocumentClient(jsonHandle)
	injectSubscriptions(client)
	db = database.NewSubscriptionsWithProvidedClient(_uuid, client)
	return db, client, _uuid
}

func NewFakeBilling() (db database.Billing, client *cosmosdb.FakeBillingDocumentClient, _uuid string) {
	_uuid = uuid.NewV4().String()
	client = cosmosdb.NewFakeBillingDocumentClient(jsonHandle)
	injectBilling(client)
	db = database.NewBillingWithProvidedClient(_uuid, client)
	return db, client, _uuid
}

func NewFakeAsyncOperations() (db database.AsyncOperations, client *cosmosdb.FakeAsyncOperationDocumentClient, _uuid string) {
	_uuid = uuid.NewV4().String()
	client = cosmosdb.NewFakeAsyncOperationDocumentClient(jsonHandle)
	db = database.NewAsyncOperationsWithProvidedClient(_uuid, client)
	return db, client, _uuid
}
