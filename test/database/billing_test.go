package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	db "github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func TestNewBilling(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	r := rand.New(rand.NewSource(time.Now().UnixMicro()))
	authorizer := cosmosdb.NewTokenAuthorizer(fmt.Sprintf("rand-%d", r.Int()))

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/dbs/testdb/colls/Billing/triggers" {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`Created`))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`Not Found`))
		}
	}))

	for _, tt := range []struct {
		name    string
		wantErr string
		dbName  string
	}{
		{
			name:    "fail: DATABASE_NAME is unset",
			wantErr: "environment variable \"DATABASE_NAME\" unset (development mode)",
		},
		{
			name:   "pass: Create new billing",
			dbName: "testdb",
		},
		{
			name:    "fail: bad endpoint",
			dbName:  "wrong-test-db",
			wantErr: "404 : ",
		},
	} {
		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", tt.dbName)
		}

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], authorizer)

		t.Run(tt.name, func(t *testing.T) {
			_, err := db.NewBilling(ctx, true, dbc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestBillingCreate(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name       string
		billingDoc *api.BillingDocument
		wantErr    string
	}{
		{
			name: "fail: document ID isn't lowercase",
			billingDoc: &api.BillingDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Create billing",
			billingDoc: &api.BillingDocument{
				ID: "lower",
			},
			wantErr: "not implemented",
		},
	} {
		billing := db.NewBillingWithProvidedClient(cosmosdb.NewFakeBillingDocumentClient(&codec.JsonHandle{}))

		t.Run(tt.name, func(t *testing.T) {
			_, err := billing.Create(ctx, tt.billingDoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestBillingGet(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name       string
		billingDoc *api.BillingDocument
		wantErr    string
		ID         string
	}{
		{
			name: "fail: document ID isn't lowercase",
			billingDoc: &api.BillingDocument{
				ID: "UPPER",
			},
			ID:      "UPPER",
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Create billing",
			billingDoc: &api.BillingDocument{
				ID: "lower",
			},
			ID: "lower",
		},
	} {
		billing, docClient := NewFakeBilling()
		doc, err := docClient.Create(ctx, "part1", tt.billingDoc, &cosmosdb.Options{})
		if err != nil {
			t.Errorf("failed to create billing documents for %s", tt.name)
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := billing.Get(ctx, doc.ID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		billing.Delete(ctx, doc)
	}
}

func TestBillingMarkForDeletion(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name           string
		wantErr        string
		billingDoc     *api.BillingDocument
		ID             string
		returnWrongDoc bool
	}{
		{
			name: "pass: Mark billing document for deletion",
			billingDoc: &api.BillingDocument{
				ID:   "my-id",
				ETag: "my-tag",
			},
			ID: "my-id",
		},
		{
			name:       "fail: 404 billing document not found",
			billingDoc: &api.BillingDocument{},
			wantErr:    "404 : ",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/Billing/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/Billing/docs":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/Billing/docs/my-id":
				buf := &bytes.Buffer{}
				err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(tt.billingDoc)
				if err != nil {
					t.Logf("\n%s\nfailed to encode billing document to request body\n%v\n", tt.name, err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("x-ms-version", "2018-12-31")
				w.Write(buf.Bytes())
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		docClient := cosmosdb.NewFakeBillingDocumentClient(&codec.JsonHandle{})
		authorizer := cosmosdb.NewTokenAuthorizer("")

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], authorizer)

		billing, err := db.NewBilling(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s: \nfailed to create new billing\n %v\n", tt.name, err)
		}

		docClient.Create(ctx, "part1", tt.billingDoc, &cosmosdb.Options{})
		_, err = billing.Create(ctx, tt.billingDoc)
		if err != nil {
			t.Errorf("\n %s\n failed to create new billing document\n%v\n", tt.name, err)
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := billing.MarkForDeletion(ctx, tt.ID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestBillingDelete(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name       string
		wantErr    string
		billingDoc *api.BillingDocument
		ID         string
	}{
		{
			name: "fail: document ID isn't lower",
			billingDoc: &api.BillingDocument{
				ID:  "1",
				Key: "UPPER",
			},
			wantErr: "key \"UPPER\" is not lower case",
		},
	} {
		billing, docClient := NewFakeBilling()
		doc, err := docClient.Create(ctx, "part1", tt.billingDoc, &cosmosdb.Options{})
		if err != nil {
			t.Errorf("failed to create billing documents for %s", tt.name)
		}

		t.Run(tt.name, func(t *testing.T) {
			err := billing.Delete(ctx, doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}
