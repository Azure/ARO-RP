package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
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

func TestNewAsyncOperations(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

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
			name:   "pass: new async operations",
			dbName: "testdb",
		},
	} {
		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", "testdb")
		}

		dbc := cosmosdb.NewDatabaseClient(log, &http.Client{}, &codec.JsonHandle{}, "https://localhost", nil)

		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAsyncOperations(ctx, true, dbc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
		os.Unsetenv("DATABASE_NAME")
	}
}

func TestAsyncOperationsCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		apDoc   *api.AsyncOperationDocument
	}{
		{
			name: "fail: ID isn't lowercase",
			apDoc: &api.AsyncOperationDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "fail: status precondition",
			apDoc: &api.AsyncOperationDocument{
				ID: "id-1",
			},
			status:  http.StatusConflict,
			wantErr: "412 : ",
		},
		{
			name: "pass: create document",
			apDoc: &api.AsyncOperationDocument{
				ID: "id-1",
			},
			status: http.StatusCreated,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/AsyncOperations/docs" {
				w.WriteHeader(tt.status)
			} else {
				t.Logf("resource requested %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)

		ap, err := NewAsyncOperations(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new asyncoperations\n%s\n", tt.name, err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := ap.Create(ctx, tt.apDoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestAsyncOperationsGet(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		ID      string
	}{
		{
			name:    "fail: ID isn't lowercase",
			ID:      "UPPER",
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Get document",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/AsyncOperations/docs/" {
				w.WriteHeader(http.StatusOK)
			} else {
				t.Logf("resource requested %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)

		ap, err := NewAsyncOperations(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new asyncoperations\n%s\n", tt.name, err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := ap.Get(ctx, tt.ID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestAsyncOperationsPatch(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		apDoc   *api.AsyncOperationDocument
		f       func(*api.AsyncOperationDocument) error
		status  int
	}{
		{
			name: "pass: replace document",
			apDoc: &api.AsyncOperationDocument{
				ID: "id-1",
			},
			f: func(*api.AsyncOperationDocument) error {
				return nil
			},
		},
		{
			name: "fail: f function return error",
			apDoc: &api.AsyncOperationDocument{
				ID: "id-1",
			},
			f: func(ap *api.AsyncOperationDocument) error {
				return fmt.Errorf("f returned error")
			},
			wantErr: "f returned error",
		},
		{
			name:   "fail: Get 404",
			status: http.StatusNotFound,
			apDoc: &api.AsyncOperationDocument{
				ID: "id-1",
			},
			f: func(*api.AsyncOperationDocument) error {
				return nil
			},
			wantErr: "404 : ",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tt.status > 0 {
				w.WriteHeader(tt.status)
			} else if r.URL.String() == "/dbs/testdb/colls/AsyncOperations/docs/"+tt.apDoc.ID {
				buf := &bytes.Buffer{}
				err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(tt.apDoc)
				if err != nil {
					t.Logf("\n%s\nfailed to encode document to request body\n%v\n", tt.name, err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("x-ms-version", "2018-12-31")
				w.Write(buf.Bytes())
			} else {
				t.Logf("resource requested %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		host := strings.SplitAfter(ts.URL, "//")
		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)

		ap, err := NewAsyncOperations(ctx, true, dbc)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new asyncoperations\n%s\n", tt.name, err.Error())
		}

		// cover NewUUID without a separate test
		tt.apDoc.ID = ap.NewUUID()

		t.Run(tt.name, func(t *testing.T) {
			_, err := ap.Patch(ctx, tt.apDoc.ID, tt.f)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}
