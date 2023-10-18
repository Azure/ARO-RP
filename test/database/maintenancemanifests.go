package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func injectMaintenanceManifests(c *cosmosdb.FakeMaintenanceManifestDocumentClient) {
	c.SetQueryHandler(database.MaintenanceManifestQueryForCluster, fakeMaintenanceManifestsQueryCluster)

	c.SetTriggerHandler("renewLease", fakeMaintenanceManifestsRenewLeaseTrigger)
}

func fakeMaintenanceManifestsQueryCluster(client cosmosdb.MaintenanceManifestDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MaintenanceManifestDocumentRawIterator {
	startingIndex, err := fakeMaintenanceManifestsGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeMaintenanceManifestDocumentErroringRawIterator(err)
	}

	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	clusterID := query.Parameters[0].Value

	fmt.Print(clusterID, startingIndex)

	var results []*api.MaintenanceManifestDocument
	for _, r := range input.MaintenanceManifestDocuments {
		if r.ClusterID == clusterID && (r.MaintenanceManifest.State == api.MaintenanceManifestStatePending ||
			r.MaintenanceManifest.State == api.MaintenanceManifestStateInProgress) {
			results = append(results, r)
		}
	}
	return cosmosdb.NewFakeMaintenanceManifestDocumentIterator(results, startingIndex)
}

func fakeMaintenanceManifestsRenewLeaseTrigger(ctx context.Context, doc *api.MaintenanceManifestDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 60
	return nil
}

func fakeMaintenanceManifestsRetryLaterTrigger(ctx context.Context, doc *api.MaintenanceManifestDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 600
	return nil
}

func fakeMaintenanceManifestsGetContinuation(options *cosmosdb.Options) (startingIndex int, err error) {
	if options != nil && options.Continuation != "" {
		startingIndex, err = strconv.Atoi(options.Continuation)
	}
	return
}
