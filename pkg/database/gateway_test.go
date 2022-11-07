package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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

func TestNewGateway(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		wantErr string
		dbName  string
		status  int
	}{
		{
			name:    "fail: DATABASE_NAME unset",
			wantErr: "environment variable \"DATABASE_NAME\" unset (development mode)",
		},
		{
			name:   "pass: Create new gateway",
			dbName: "testdb",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		}))

		host := strings.SplitAfter(ts.URL, "//")

		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)

		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", tt.dbName)
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGateway(ctx, true, dbc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		os.Unsetenv("DATABASE_NAME")
	}
}

func TestGatewayCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.GatewayDocument
	}{
		{
			name: "fail: ID isn't lowercase",
			doc: &api.GatewayDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Create document",
			doc: &api.GatewayDocument{
				ID: "id-1",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Gateway/docs" {
				w.WriteHeader(http.StatusCreated)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestGateway(ctx, log, ts, t)

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Create(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestGatewayDelete(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.GatewayDocument
	}{
		{
			name: "fail: ID isn't lowercase",
			doc: &api.GatewayDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Delete document",
			doc: &api.GatewayDocument{
				ID: "id-1",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Gateway/docs/"+tt.doc.ID {
				w.WriteHeader(http.StatusNoContent)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestGateway(ctx, log, ts, t)

		t.Run(tt.name, func(t *testing.T) {
			err := c.Delete(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestGatewayGet(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.GatewayDocument
	}{
		{
			name: "fail: ID isn't lowercase",
			doc: &api.GatewayDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Get document",
			doc: &api.GatewayDocument{
				ID: "id-1",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Gateway/docs/"+tt.doc.ID {
				w.WriteHeader(http.StatusOK)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestGateway(ctx, log, ts, t)

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Get(ctx, tt.doc.ID)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestGatewayPatch(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	doc := &api.GatewayDocument{
		ID: "id",
	}

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		doc     *api.GatewayDocument
		newDoc  *api.GatewayDocument
		f       func(*api.GatewayDocument) error
	}{
		{
			name:    "fail: 404 not found",
			status:  http.StatusNotFound,
			doc:     doc,
			wantErr: "404 : ",
		},
		{
			name: "fail: f returns error",
			f: func(*api.GatewayDocument) error {
				return fmt.Errorf("error from f")
			},
			status:  http.StatusOK,
			doc:     doc,
			wantErr: "error from f",
		},
		{
			name: "fail: ID isn't lowercase",
			f: func(*api.GatewayDocument) error {
				return nil
			},
			doc: doc,
			newDoc: &api.GatewayDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Patch document",
			f: func(*api.GatewayDocument) error {
				return nil
			},
			doc:    doc,
			newDoc: doc,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Gateway/docs/"+tt.doc.ID {
				if tt.status > 0 {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, tt.newDoc)
				}
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestGateway(ctx, log, ts, t)

		// Cover without needing separate tests
		c.NewUUID()
		c.ChangeFeed()

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Patch(ctx, tt.doc.ID, tt.f)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func newTestGateway(ctx context.Context, log *logrus.Entry, ts *httptest.Server, t *testing.T) Gateway {
	host := strings.SplitAfter(ts.URL, "//")
	dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
	c, err := NewGateway(ctx, true, dbc)
	if err != nil {
		t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
	}
	return c
}
