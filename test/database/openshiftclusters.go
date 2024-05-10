package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type ByKey []*api.OpenShiftClusterDocument

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return strings.Compare(a[i].Key, a[j].Key) < 0 }

func getQueuedOpenShiftDocuments(client cosmosdb.OpenShiftClusterDocumentClient) (res []*api.OpenShiftClusterDocument, err error) {
	docs, err := fakeOpenShiftClustersGetAllDocuments(client)
	if err != nil {
		return nil, err
	}

	for _, r := range docs {
		var include bool
		switch r.OpenShiftCluster.Properties.ProvisioningState {
		case
			api.ProvisioningStateCreating,
			api.ProvisioningStateUpdating,
			api.ProvisioningStateAdminUpdating,
			api.ProvisioningStateDeleting:
			include = true
		}

		if include && (r.LeaseExpires > 0 && int64(r.LeaseExpires) < time.Now().Unix()) {
			include = false
		}
		if include {
			res = append(res, r)
		}
	}
	return
}

func fakeOpenShiftClustersQueueLengthQuery(client cosmosdb.OpenShiftClusterDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.OpenShiftClusterDocumentRawIterator {
	results, err := getQueuedOpenShiftDocuments(client)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}
	return &fakeOpenShiftClustersQueueLengthIterator{resultCount: len(results)}
}

func fakeOpenShiftClustersDequeueQuery(client cosmosdb.OpenShiftClusterDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.OpenShiftClusterDocumentRawIterator {
	docs, err := getQueuedOpenShiftDocuments(client)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}
	return cosmosdb.NewFakeOpenShiftClusterDocumentIterator(docs, 0)
}

func fakeOpenshiftClustersMatchQuery(client cosmosdb.OpenShiftClusterDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.OpenShiftClusterDocumentRawIterator {
	var results []*api.OpenShiftClusterDocument

	startingIndex, err := fakeOpenShiftClustersGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}

	docs, err := fakeOpenShiftClustersGetAllDocuments(client)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}
	for _, r := range docs {
		var key string
		switch query.Parameters[0].Name {
		case "@key":
			key = r.Key
		case "@clientID":
			key = r.ClientIDKey
		case "@resourceGroupID":
			key = r.ClusterResourceGroupIDKey
		default:
			return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(cosmosdb.ErrNotImplemented)
		}
		if key == query.Parameters[0].Value {
			results = append(results, r)
		}
	}
	return cosmosdb.NewFakeOpenShiftClusterDocumentIterator(results, startingIndex)
}

func fakeOpenShiftClustersGetAllDocuments(client cosmosdb.OpenShiftClusterDocumentClient) ([]*api.OpenShiftClusterDocument, error) {
	input, err := client.ListAll(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	docs := input.OpenShiftClusterDocuments
	sort.Sort(ByKey(docs))
	return docs, nil
}

func fakeOpenShiftClustersGetContinuation(options *cosmosdb.Options) (startingIndex int, err error) {
	if options != nil && options.Continuation != "" {
		startingIndex, err = strconv.Atoi(options.Continuation)
	}
	return
}

func fakeOpenshiftClustersPrefixQuery(client cosmosdb.OpenShiftClusterDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.OpenShiftClusterDocumentRawIterator {
	startingIndex, err := fakeOpenShiftClustersGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}

	docs, err := fakeOpenShiftClustersGetAllDocuments(client)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}
	var results []*api.OpenShiftClusterDocument
	for _, r := range docs {
		if strings.Index(r.Key, query.Parameters[0].Value) == 0 {
			results = append(results, r)
		}
	}

	return cosmosdb.NewFakeOpenShiftClusterDocumentIterator(results, startingIndex)
}

func fakeOpenShiftClustersRenewLeaseTrigger(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	doc.LeaseExpires = int(time.Now().Unix()) + 60
	return nil
}

func openShiftClusterConflictChecker(one *api.OpenShiftClusterDocument, two *api.OpenShiftClusterDocument) bool {
	if one.ClusterResourceGroupIDKey != "" && two.ClusterResourceGroupIDKey != "" && one.ClusterResourceGroupIDKey == two.ClusterResourceGroupIDKey {
		return true
	}
	if one.ClientIDKey != "" && two.ClientIDKey != "" && one.ClientIDKey == two.ClientIDKey {
		return true
	}
	return false
}

func fakeOpenShiftClustersOnlyResourceID(client cosmosdb.OpenShiftClusterDocumentClient, query *cosmosdb.Query, options *cosmosdb.Options) cosmosdb.OpenShiftClusterDocumentRawIterator {
	startingIndex, err := fakeOpenShiftClustersGetContinuation(options)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}

	docs, err := fakeOpenShiftClustersGetAllDocuments(client)
	if err != nil {
		return cosmosdb.NewFakeOpenShiftClusterDocumentErroringRawIterator(err)
	}

	newDocs := make([]*api.OpenShiftClusterDocument, 0)

	for _, d := range docs {

		newDocs = append(newDocs, &api.OpenShiftClusterDocument{
			Key: d.Key,
		})

	}

	return cosmosdb.NewFakeOpenShiftClusterDocumentIterator(newDocs, startingIndex)
}

func injectOpenShiftClusters(c *cosmosdb.FakeOpenShiftClusterDocumentClient) {
	c.SetQueryHandler(database.OpenShiftClustersDequeueQuery, fakeOpenShiftClustersDequeueQuery)
	c.SetQueryHandler(database.OpenShiftClustersQueueLengthQuery, fakeOpenShiftClustersQueueLengthQuery)
	c.SetQueryHandler(database.OpenShiftClustersGetQuery, fakeOpenshiftClustersMatchQuery)
	c.SetQueryHandler(database.OpenshiftClustersClientIdQuery, fakeOpenshiftClustersMatchQuery)
	c.SetQueryHandler(database.OpenshiftClustersResourceGroupQuery, fakeOpenshiftClustersMatchQuery)
	c.SetQueryHandler(database.OpenshiftClustersPrefixQuery, fakeOpenshiftClustersPrefixQuery)
	c.SetQueryHandler(database.OpenshiftClustersClusterResourceIDOnlyQuery, fakeOpenShiftClustersOnlyResourceID)

	c.SetTriggerHandler("renewLease", fakeOpenShiftClustersRenewLeaseTrigger)

	c.SetSorter(func(in []*api.OpenShiftClusterDocument) { sort.Sort(ByKey(in)) })
	c.SetConflictChecker(openShiftClusterConflictChecker)
}

// fakeOpenShiftClustersQueueLengthIterator is a RawIterator that will produce a
// document containing a list of a single integer when NextRaw is called.
type fakeOpenShiftClustersQueueLengthIterator struct {
	called      bool
	resultCount int
}

func (i *fakeOpenShiftClustersQueueLengthIterator) Next(ctx context.Context, maxItemCount int) (*api.OpenShiftClusterDocuments, error) {
	return nil, cosmosdb.ErrNotImplemented
}

func (i *fakeOpenShiftClustersQueueLengthIterator) NextRaw(ctx context.Context, continuation int, out interface{}) error {
	if i.called {
		return errors.New("can't call twice")
	}
	i.called = true

	res := fmt.Sprintf(`{"Count": 1, "Documents": [%d]}`, i.resultCount)
	return json.NewDecoder(bytes.NewBufferString(res)).Decode(out)
}

func (i *fakeOpenShiftClustersQueueLengthIterator) Continuation() string {
	return ""
}

func GetResourcePath(subscriptionID string, resourceID string) string {
	return fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscriptionID, resourceID)
}

func GetPreflightPath(subscriptionID string, deploymentID string) string {
	return fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/deployments/%s/preflight", subscriptionID, deploymentID)
}
