package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

const (
	billingContainerName string = "Billing"
)

type billing struct {
	c cosmosdb.BillingDocumentClient
}

// Billing is the database interface for BillingDocuments
type Billing interface {
	Create(context.Context, *api.BillingDocument) (*api.BillingDocument, error)
	Get(context.Context, string) (*api.BillingDocument, error)
	MarkForDeletion(context.Context, string) (*api.BillingDocument, error)
	UpdateLastBillingTimestamp(context.Context, string, int) (*api.BillingDocument, error)
	List(string) cosmosdb.BillingDocumentIterator
	ListAll(context.Context) (*api.BillingDocuments, error)
	Delete(context.Context, *api.BillingDocument) error
}

// NewBilling returns a new Billing
func NewBilling(ctx context.Context, dbc cosmosdb.DatabaseClient, dbName string, sqlResourceClient *armcosmos.SQLResourcesClient, location, resourceGroup, dbAccountName string) (Billing, error) {
	collc := cosmosdb.NewCollectionClient(dbc, dbName)

	triggerResources := []*armcosmos.SQLTriggerResource{
		{
			ID: to.Ptr("setCreationBillingTimeStamp"),
			Body: to.Ptr(`function trigger() {
				var request = getContext().getRequest();
				var body = request.getBody();
				var date = new Date();
				var now = Math.floor(date.getTime() / 1000);
				var billingBody = body["billing"];
				if (!billingBody["creationTime"]) {
					billingBody["creationTime"] = now;
				}
				request.setBody(body);
			}`),
			TriggerOperation: to.Ptr(armcosmos.TriggerOperation("Create")),
			TriggerType:      to.Ptr(armcosmos.TriggerType("Pre")),
		},
		{
			ID: to.Ptr("setDeletionBillingTimeStamp"),
			Body: to.Ptr(`function trigger() {
				var request = getContext().getRequest();
				var body = request.getBody();
				var date = new Date();
				var now = Math.floor(date.getTime() / 1000);
				var billingBody = body["billing"];
				if (!billingBody["creationTime"]) {
					billingBody["creationTime"] = now;
				}
				request.setBody(body);
			}`),
			TriggerOperation: to.Ptr(armcosmos.TriggerOperation("Replace")),
			TriggerType:      to.Ptr(armcosmos.TriggerType("Pre")),
		},
	}

	for _, triggerResource := range triggerResources {
		createUpdateSQLTriggerParameters := armcosmos.SQLTriggerCreateUpdateParameters{
			Properties: &armcosmos.SQLTriggerCreateUpdateProperties{
				Options:  &armcosmos.CreateUpdateOptions{},
				Resource: triggerResource,
			},
			Location: &location,
		}
		ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
		defer cancel()

		poller, err := sqlResourceClient.BeginCreateUpdateSQLTrigger(ctx, resourceGroup, dbAccountName, dbName, billingContainerName, *triggerResource.ID, createUpdateSQLTriggerParameters, nil)
		if err != nil {
			return nil, err
		}
		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	documentClient := cosmosdb.NewBillingDocumentClient(collc, collBilling)
	return NewBillingWithProvidedClient(documentClient), nil
}

func NewBillingWithProvidedClient(client cosmosdb.BillingDocumentClient) Billing {
	return &billing{
		c: client,
	}
}

// Creating Billing Document
func (c *billing) Create(ctx context.Context, doc *api.BillingDocument) (*api.BillingDocument, error) {
	if doc.ID != strings.ToLower(doc.ID) {
		return nil, fmt.Errorf("id %q is not lower case", doc.ID)
	}

	return c.c.Create(ctx, doc.ID, doc, &cosmosdb.Options{PreTriggers: []string{"setCreationBillingTimeStamp"}})
}

func (c *billing) Get(ctx context.Context, id string) (*api.BillingDocument, error) {
	if id != strings.ToLower(id) {
		return nil, fmt.Errorf("id %q is not lower case", id)
	}

	return c.c.Get(ctx, id, id, nil)
}

func (c *billing) patch(ctx context.Context, id string, f func(*api.BillingDocument) error, options *cosmosdb.Options) (*api.BillingDocument, error) {
	var doc *api.BillingDocument

	err := cosmosdb.RetryOnPreconditionFailed(func() (err error) {
		doc, err = c.Get(ctx, id)
		if err != nil {
			return
		}

		err = f(doc)
		if err != nil {
			return
		}

		doc, err = c.c.Replace(ctx, doc.ID, doc, options)
		return
	})

	return doc, err
}

// MarkForDeletion update the deletion timestamp field in the document
func (c *billing) MarkForDeletion(ctx context.Context, id string) (*api.BillingDocument, error) {
	return c.patch(ctx, id, func(billingdoc *api.BillingDocument) error {
		return nil
	}, &cosmosdb.Options{PreTriggers: []string{"setDeletionBillingTimeStamp"}})
}

// List produces and iterator for paging through all billing documents.
func (c *billing) List(continuation string) cosmosdb.BillingDocumentIterator {
	return c.c.List(&cosmosdb.Options{Continuation: continuation})
}

// ListAll list all the billing documents
func (c *billing) ListAll(ctx context.Context) (*api.BillingDocuments, error) {
	return c.c.ListAll(ctx, nil)
}

// Delete a billing document
func (c *billing) Delete(ctx context.Context, doc *api.BillingDocument) error {
	if doc.Key != strings.ToLower(doc.Key) {
		return fmt.Errorf("key %q is not lower case", doc.Key)
	}

	return c.c.Delete(ctx, doc.ID, doc, &cosmosdb.Options{NoETag: true})
}

// UpdateLastBillingTimestamp update the last billing timestamp field in the document with the time provided
// This time will be provided by the billing service so we don't need to use trigger
func (c *billing) UpdateLastBillingTimestamp(ctx context.Context, id string, time int) (*api.BillingDocument, error) {
	return c.patch(ctx, id, func(billingdoc *api.BillingDocument) error {
		billingdoc.Billing.LastBillingTime = time
		return nil
	}, nil)
}
