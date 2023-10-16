package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func injectMaintenanceManifests(c *cosmosdb.FakeMaintenanceManifestDocumentClient) {
	c.SetQueryHandler(database.MaintenanceManifestQueryForCluster, fakeMaintenanceManifestsQueryCluster)

	c.SetTriggerHandler("renewLease", fakeMaintenanceManifestsRenewLeaseTrigger)
	c.SetTriggerHandler("retryLater", fakeMaintenanceManifestsRetryLaterTrigger)
}

func getQueuedMaintenanceManifestDocuments(client cosmosdb.MaintenanceManifestDocumentClient, clusterID string) (results []*api.MaintenanceManifestDocument) {
	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	fmt.Println(input)
	for _, r := range input.MaintenanceManifestDocuments {
		fmt.Println(r)
		if r.ClusterID == clusterID && (r.MaintenanceManifest.State == api.MaintenanceManifestStatePending ||
			r.MaintenanceManifest.State == api.MaintenanceManifestStateInProgress) {
			results = append(results, r)
		}
	}
	return
}

func fakeMaintenanceManifestsQueryCluster(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceManifestDocumentRawIterator {
	docs := getQueuedMaintenanceManifestDocuments(client, query.Parameters[0].Value)
	return cosmosdb.NewFakeMaintenanceManifestDocumentIterator(docs, 0)
}

func fakeMaintenanceManifestsRenewLeaseTrigger(ctx context.Context, doc *api.MaintenanceManifestDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 60
	return nil
}

func fakeMaintenanceManifestsRetryLaterTrigger(ctx context.Context, doc *api.MaintenanceManifestDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 600
	return nil
}
