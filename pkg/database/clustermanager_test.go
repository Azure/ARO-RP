package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func TestNewClusterManagerConfigurations(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		wantErr string
		dbName  string
	}{
		{
			name:   "pass: New Cluster Manager Configurations",
			dbName: "testdb",
		},
		{
			name:    "fail: unset DATABASE_NAME",
			wantErr: "environment variable \"DATABASE_NAME\" unset (development mode)",
		},
	} {
		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", "testdb")
		}

		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)

		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClusterManagerConfigurations(ctx, true, dbc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		os.Unsetenv("DATABASE_NAME")
	}
}

func TestClusterManagerConfigurationsCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		cDoc    *api.ClusterManagerConfigurationDocument
		status  int
	}{
		{
			name: "fail: uppercase document ID",
			cDoc: &api.ClusterManagerConfigurationDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "fail: status conflict",
			cDoc: &api.ClusterManagerConfigurationDocument{
				ID:  "id-1",
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			wantErr: "412 : ",
			status:  http.StatusConflict,
		},
		{
			name: "fail: invalid key format",
			cDoc: &api.ClusterManagerConfigurationDocument{
				ID:  "id-1",
				Key: "invalid/format",
			},
			wantErr: "parsing failed for invalid/format. Invalid resource Id format",
			status:  http.StatusCreated,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/ClusterManagerConfigurations/docs" {
				w.Header().Set(`Content-Type`, `application/json`)
				w.WriteHeader(tt.status)
			} else {
				t.Logf("resource %s not found", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
		cm, err := NewClusterManagerConfigurations(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\nfailed to create cluster manager configurations\n%s\n", tt.name, err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := cm.Create(ctx, tt.cDoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestClusterManagerConfigurationsGet(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		cDocs   *api.ClusterManagerConfigurationDocuments
		ID      string
		status  int
	}{
		{
			name:    "fail: uppercase id",
			ID:      "UPPER",
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name:    "fail: invalid partition key",
			ID:      "invalid/partition/key",
			wantErr: "parsing failed for invalid/partition/key. Invalid resource Id format",
		},
		{
			name:    "fail: 404 not found",
			ID:      "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			wantErr: "404 : ",
		},
		{
			name: "fail: Expected one document",
			ID:   "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			cDocs: &api.ClusterManagerConfigurationDocuments{
				Count:      2,
				ResourceID: "id-1",
				ClusterManagerConfigurationDocuments: []*api.ClusterManagerConfigurationDocument{
					{
						ID:         "doc-1",
						ResourceID: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
						ETag:       "tag",
					},
					{
						ID:         "doc-2",
						ResourceID: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
						ETag:       "tag",
					},
				},
			},
			wantErr: "read 2 documents, expected <= 1",
		},
		{
			name:    "fail: incorrect response status",
			ID:      "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			status:  http.StatusNotFound,
			wantErr: "404 : ",
		},
		{
			name: "pass: Get document",
			ID:   "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			cDocs: &api.ClusterManagerConfigurationDocuments{
				Count:      1,
				ResourceID: "id-1",
				ClusterManagerConfigurationDocuments: []*api.ClusterManagerConfigurationDocument{
					{
						ID:         "doc-1",
						ResourceID: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
						ETag:       "tag",
					},
				},
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tt.status > 0 {
				w.WriteHeader(tt.status)
			} else if r.URL.String() == "/dbs/testdb/colls/ClusterManagerConfigurations/docs" {
				buf := &bytes.Buffer{}
				err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(tt.cDocs)
				if err != nil {
					t.Logf("\n%s\nfailed to encode document to request body\n%v\n", tt.name, err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("x-ms-version", "2018-12-31")
				w.Write(buf.Bytes())
			} else {
				t.Logf("Resource %s not found", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
		cm, err := NewClusterManagerConfigurations(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\nfailed to create cluster manager configurations\n%s\n", tt.name, err.Error())
		}

		// Use NewUUID here to cover function test coverage
		if tt.cDocs != nil {
			tt.cDocs.ClusterManagerConfigurationDocuments[0].ID = cm.NewUUID()
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := cm.Get(ctx, tt.ID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestClusterConfigurationReplace(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		cDoc    *api.ClusterManagerConfigurationDocument
	}{
		{
			name: "fail: uppercase id",
			cDoc: &api.ClusterManagerConfigurationDocument{
				ID:           "UPPER",
				PartitionKey: "my/doc",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: update document",
			cDoc: &api.ClusterManagerConfigurationDocument{
				ID:           "id-1",
				PartitionKey: "my/doc",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/ClusterManagerConfigurations/docs/"+tt.cDoc.ID {
				w.WriteHeader(http.StatusOK)
			}
		}))
		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
		cm, err := NewClusterManagerConfigurations(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\nfailed to create cluster manager configurations\n%s\n", tt.name, err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := cm.Update(ctx, tt.cDoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestClusterConfigurationDelete(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		cDoc    *api.ClusterManagerConfigurationDocument
	}{
		{
			name: "fail: uppercase id",
			cDoc: &api.ClusterManagerConfigurationDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Delete document",
			cDoc: &api.ClusterManagerConfigurationDocument{
				ID: "id-1",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/ClusterManagerConfigurations/docs/"+tt.cDoc.ID {
				w.WriteHeader(http.StatusNoContent)
			}
		}))
		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
		cm, err := NewClusterManagerConfigurations(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\nfailed to create cluster manager configurations\n%s\n", tt.name, err.Error())
		}

		// Cover ChangeFeed in test coverage without a separate test
		cm.ChangeFeed()

		t.Run(tt.name, func(t *testing.T) {
			err := cm.Delete(ctx, tt.cDoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}
