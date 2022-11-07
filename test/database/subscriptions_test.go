package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	db "github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func TestNewSubscriptions(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name        string
		wantErr     string
		dbName      string
		status      int
		failPrecond bool
	}{
		{
			name:   "pass: new subscriptions",
			dbName: "testdb",
			status: http.StatusCreated,
		},
		{
			name:    "fail: DATABASE_NAME is unset",
			wantErr: "environment variable \"DATABASE_NAME\" unset (development mode)",
			status:  http.StatusCreated,
		},
		{
			name:        "fail: http precondition status",
			dbName:      "testdb",
			status:      http.StatusPreconditionFailed,
			wantErr:     "412 : ",
			failPrecond: true,
		},
	} {
		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", "testdb")
		}

		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Subscriptions/triggers" {
				if tt.failPrecond {
					w.Header().Set(`Content-Type`, `application/json`)
					w.WriteHeader(tt.status)
				} else {
					w.WriteHeader(tt.status)
				}
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		r := rand.New(rand.NewSource(time.Now().UnixMicro()))
		authorizer := cosmosdb.NewTokenAuthorizer(fmt.Sprintf("rand-%d", r.Int()))

		host := strings.SplitAfter(ts.URL, "//")

		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], authorizer)

		t.Run(tt.name, func(t *testing.T) {
			_, err := db.NewSubscriptions(ctx, true, dbc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		os.Unsetenv("DATABASE_NAME")
	}
}

func TestSubscriptionUpdate(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name    string
		ID      string
		wantErr string
		subDoc  *api.SubscriptionDocument
	}{
		{
			name: "changefeed isn't nil",
			subDoc: &api.SubscriptionDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
	} {
		d := cosmosdb.NewFakeSubscriptionDocumentClient(&codec.JsonHandle{})
		subscriptions := db.NewSubscriptionsWithProvidedClient(d, "")

		if tt.subDoc != nil {
			d.Create(ctx, "part1", tt.subDoc, &cosmosdb.Options{})
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := subscriptions.Update(ctx, tt.subDoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriptionChangeFeed(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		ID      string
		wantErr string
		subDoc  *api.SubscriptionDocument
	}{
		{
			name: "changefeed isn't nil",
		},
	} {
		d := cosmosdb.NewFakeSubscriptionDocumentClient(&codec.JsonHandle{})
		subscriptions := db.NewSubscriptionsWithProvidedClient(d, "")

		if tt.subDoc != nil {
			d.Create(ctx, "part1", tt.subDoc, &cosmosdb.Options{})
		}
		t.Run(tt.name, func(t *testing.T) {
			docIterator := subscriptions.ChangeFeed()
			if docIterator == nil {
				t.Errorf("Document Iterator should not be nil")
			}
		})
	}
}

func TestSubscriptionCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name        string
		wantErr     string
		devMode     bool
		subDoc      *api.SubscriptionDocument
		status      int
		failPrecond bool
	}{
		{
			name:    "pass: create",
			devMode: true,
			subDoc:  &api.SubscriptionDocument{},
			status:  http.StatusCreated,
		},
		{
			name:    "fail: ID isn't lowercase",
			devMode: true,
			subDoc: &api.SubscriptionDocument{
				ID: "UPPERCASE",
			},
			wantErr: "id \"UPPERCASE\" is not lower case",
			status:  http.StatusCreated,
		},
		{
			name:    "fail: Wrong status type 200",
			subDoc:  &api.SubscriptionDocument{},
			status:  http.StatusOK,
			wantErr: "200 : ",
		},
		{
			name:        "fail: status precondition failed",
			subDoc:      &api.SubscriptionDocument{},
			status:      http.StatusConflict,
			wantErr:     "412 : ",
			failPrecond: true,
		},
	} {
		t.Setenv("DATABASE_NAME", "testdb")
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Subscriptions/docs" {
				if tt.failPrecond {
					w.Header().Set(`Content-Type`, `application/json`)
					w.WriteHeader(tt.status)
				} else {
					w.WriteHeader(tt.status)
				}
			} else if r.URL.String() == "/dbs/testdb/colls/Subscriptions/triggers" {
				w.WriteHeader(http.StatusCreated)
			} else {
				t.Logf("Resource not found %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		r := rand.New(rand.NewSource(time.Now().UnixMicro()))
		authorizer := cosmosdb.NewTokenAuthorizer(fmt.Sprintf("rand-%d", r.Int()))

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], authorizer)

		subscriptions, err := db.NewSubscriptions(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new subscriptions, %s\n", tt.name, err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := subscriptions.Create(ctx, tt.subDoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriptionGet(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name    string
		ID      string
		wantErr string
		subdoc  *api.SubscriptionDocument
	}{
		{
			name:    "fail: id isn't lowercase",
			ID:      "UPPERCASE",
			wantErr: "id \"UPPERCASE\" is not lower case",
		},
		{
			name:    "fail: no billing document id found",
			ID:      "my-id",
			wantErr: "404 : ",
		},
		{
			name: "pass: Get",
			ID:   "my-id",
			subdoc: &api.SubscriptionDocument{
				ID: "my-id",
			},
		},
	} {
		doc := cosmosdb.NewFakeSubscriptionDocumentClient(&codec.JsonHandle{})
		subscriptions := db.NewSubscriptionsWithProvidedClient(doc, "")

		if tt.subdoc != nil {
			doc.Create(ctx, "part1", tt.subdoc, &cosmosdb.Options{})
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := subscriptions.Get(ctx, tt.ID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriptionLease(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name     string
		ID       string
		wantErr  string
		subDoc   *api.SubscriptionDocument
		newLease string
	}{
		{
			name:    "fail: no document id exists",
			ID:      "my-id",
			wantErr: "404 : ",
		},
		{
			name: "fail: lost lease",
			subDoc: &api.SubscriptionDocument{
				ID: "my-id",
			},
			ID:       "my-id",
			wantErr:  "lost lease",
			newLease: "new-uuid",
		},
	} {
		doc := cosmosdb.NewFakeSubscriptionDocumentClient(&codec.JsonHandle{})
		subscriptions := db.NewSubscriptionsWithProvidedClient(doc, "initial-uuid")

		if tt.subDoc != nil {
			doc.Create(ctx, "part1", tt.subDoc, &cosmosdb.Options{})
			if tt.newLease != "" {
				tt.subDoc.LeaseOwner = tt.newLease
			}
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := subscriptions.Lease(ctx, tt.ID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriptionEndLease(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		ID      string
		wantErr string
		subDoc  *api.SubscriptionDocument
		done    bool
	}{
		{
			name: "pass: end lease",
			subDoc: &api.SubscriptionDocument{
				ID: "my-id",
			},
			ID:      "my-id",
			wantErr: "not implemented",
			done:    false,
		},
		{
			name: "pass: done = true",
			subDoc: &api.SubscriptionDocument{
				ID: "my-id",
			},
			ID:      "my-id",
			wantErr: "not implemented",
			done:    true,
		},
	} {
		docClient := cosmosdb.NewFakeSubscriptionDocumentClient(&codec.JsonHandle{})
		subscriptions := db.NewSubscriptionsWithProvidedClient(docClient, "")

		if tt.subDoc != nil {
			docClient.Create(ctx, "part1", tt.subDoc, &cosmosdb.Options{})
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := subscriptions.EndLease(ctx, tt.ID, tt.done, true)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriptionDequeue(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name    string
		wantErr string
		subDocs []*api.SubscriptionDocument
	}{
		{
			name: "pass: no documents to dequeue",
		},
		{
			name: "pass: dequeue documents",
			subDocs: []*api.SubscriptionDocument{
				{
					ID:       "1",
					Deleting: true,
				},
				{
					ID:       "2",
					Deleting: true,
				},
				{
					ID:       "3",
					Deleting: true,
				},
			},
		},
		{
			name:    "fail: cannot iterate",
			wantErr: "not implemented",
		},
	} {
		var subscriptions db.Subscriptions
		var docClient *cosmosdb.FakeSubscriptionDocumentClient
		if tt.wantErr != "" {
			subscriptions = db.NewSubscriptionsWithProvidedClient(cosmosdb.NewFakeSubscriptionDocumentClient(&codec.JsonHandle{}), "")
		} else {
			subscriptions, docClient = NewFakeSubscriptions()
		}

		if tt.subDocs != nil {
			for _, doc := range tt.subDocs {
				docClient.Create(ctx, "part1", doc, &cosmosdb.Options{})
			}
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := subscriptions.Dequeue(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}
