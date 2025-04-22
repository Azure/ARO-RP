package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/test/util/deterministicuuid"
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
	uuid := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.CLUSTERS)
	coll := &fakeCollectionClient{}
	client = cosmosdb.NewFakeOpenShiftClusterDocumentClient(jsonHandle)
	injectOpenShiftClusters(client)
	db = database.NewOpenShiftClustersWithProvidedClient(client, coll, "", uuid)
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
	uuid := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.ASYNCOPERATIONS)
	client = cosmosdb.NewFakeAsyncOperationDocumentClient(jsonHandle)
	db = database.NewAsyncOperationsWithProvidedClient(client, uuid)
	return db, client
}

func NewFakePortal() (db database.Portal, client *cosmosdb.FakePortalDocumentClient) {
	uuid := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.PORTAL)
	client = cosmosdb.NewFakePortalDocumentClient(jsonHandle)
	db = database.NewPortalWithProvidedClient(client, uuid)
	return db, client
}

func NewFakeGateway() (db database.Gateway, client *cosmosdb.FakeGatewayDocumentClient) {
	uuid := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.GATEWAY)
	client = cosmosdb.NewFakeGatewayDocumentClient(jsonHandle)
	db = database.NewGatewayWithProvidedClient(client, uuid)
	return db, client
}

func NewFakeOpenShiftVersions(uuid uuid.Generator) (db database.OpenShiftVersions, client *cosmosdb.FakeOpenShiftVersionDocumentClient) {
	client = cosmosdb.NewFakeOpenShiftVersionDocumentClient(jsonHandle)
	db = database.NewOpenShiftVersionsWithProvidedClient(client, uuid)
	return db, client
}

func NewFakePlatformWorkloadIdentityRoleSets(uuid uuid.Generator) (db database.PlatformWorkloadIdentityRoleSets, client *cosmosdb.FakePlatformWorkloadIdentityRoleSetDocumentClient) {
	client = cosmosdb.NewFakePlatformWorkloadIdentityRoleSetDocumentClient(jsonHandle)
	db = database.NewPlatformWorkloadIdentityRoleSetsWithProvidedClient(client, uuid)
	return db, client
}

func NewFakeMaintenanceManifests(now func() time.Time) (db database.MaintenanceManifests, client *cosmosdb.FakeMaintenanceManifestDocumentClient) {
	uuid := deterministicuuid.NewTestUUIDGenerator(deterministicuuid.MAINTENANCE_MANIFESTS)
	coll := &fakeCollectionClient{}
	client = cosmosdb.NewFakeMaintenanceManifestDocumentClient(jsonHandle)
	injectMaintenanceManifests(client, now)
	db = database.NewMaintenanceManifestsWithProvidedClient(client, coll, "", uuid)
	return db, client
}
