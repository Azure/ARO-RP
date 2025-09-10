package database

import (
	"context"
	"slices"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func getQueuedMonitorDocuments(client cosmosdb.MonitorDocumentClient) (results []*api.MonitorDocument) {
	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		// TODO: should this never happen?
		panic(err)
	}

	for _, r := range input.MonitorDocuments {
		if int64(r.LeaseExpires) < time.Now().Unix() {
			results = append(results, r)
		}
	}
	return
}

func fakeMonitorsDequeueQuery(client cosmosdb.MonitorDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.MonitorDocumentRawIterator {
	docs := getQueuedMonitorDocuments(client)
	return cosmosdb.NewFakeMonitorDocumentIterator(docs, 0)
}

func fakeMonitoringRenewLeaseTrigger(ctx context.Context, doc *api.MonitorDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 60
	return nil
}

func fakeMonitorRetryLaterTrigger(ctx context.Context, doc *api.MonitorDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 600
	return nil
}

func fakeMonitorGetMasterQuery(client cosmosdb.MonitorDocumentClient, query *cosmosdb.Query, opts *cosmosdb.Options) cosmosdb.MonitorDocumentRawIterator {
	doc, _ := client.Get(context.Background(), "", "master", nil)
	out := []*api.MonitorDocument{}

	if time.Unix(int64(doc.LeaseExpires), 0).Before(time.Now()) {
		out = append(out, doc)
	}

	return cosmosdb.NewFakeMonitorDocumentIterator(out, 0)
}

func fakeMonitorGetAllButMasterHandler(client cosmosdb.MonitorDocumentClient, query *cosmosdb.Query, opts *cosmosdb.Options) cosmosdb.MonitorDocumentRawIterator {
	docs, _ := client.ListAll(context.TODO(), nil)

	if docs == nil {
		return cosmosdb.NewFakeMonitorDocumentIterator(nil, 0)
	}

	remainingDocs := slices.DeleteFunc(docs.MonitorDocuments, func(d *api.MonitorDocument) bool {
		return d.ID == "master"
	})
	return cosmosdb.NewFakeMonitorDocumentIterator(remainingDocs, 0)
}

func injectMonitors(c *cosmosdb.FakeMonitorDocumentClient) {
	c.SetQueryHandler(database.SubscriptionsDequeueQuery, fakeMonitorsDequeueQuery)
	c.SetQueryHandler(`SELECT * FROM Monitors doc WHERE doc.id = "master" AND (doc.leaseExpires ?? 0) < GetCurrentTimestamp() / 1000`, fakeMonitorGetMasterQuery)
	c.SetQueryHandler(`SELECT * FROM Monitors doc WHERE doc.id != "master"`, fakeMonitorGetAllButMasterHandler)

	c.SetTriggerHandler("renewLease", fakeMonitoringRenewLeaseTrigger)
	c.SetTriggerHandler("retryLater", fakeMonitorRetryLaterTrigger)
}
