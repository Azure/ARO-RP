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

func TestNewPortal(t *testing.T) {
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
			name:   "pass: Create new Portal",
			dbName: "testdb",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", tt.dbName)
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestPortal(ctx, log, ts, t)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		os.Unsetenv("DATABASE_NAME")
	}
}

func TestPortalCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.PortalDocument
		status  int
	}{
		{
			name: "fail: ID isn't lowercase",
			doc: &api.PortalDocument{
				ID: "UPPER",
			},
			status:  http.StatusCreated,
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "fail: Status conflict",
			doc: &api.PortalDocument{
				ID: "id",
			},
			status:  http.StatusConflict,
			wantErr: "412 : ",
		},
		{
			name: "pass: Create new Portal",
			doc: &api.PortalDocument{
				ID: "id",
			},
			status: http.StatusCreated,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Portal/docs" {
				w.WriteHeader(tt.status)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestPortal(ctx, log, ts, t)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new Portal: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Create(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestPortalPatch(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.PortalDocument
		status  int
		f       func(*api.PortalDocument) error
	}{
		{
			// Covers Get
			name: "fail: ID isn't lowercase",
			doc: &api.PortalDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
			f: func(*api.PortalDocument) error {
				return nil
			},
		},
		{
			name: "fail: Status conflict",
			doc: &api.PortalDocument{
				ID: "id",
			},
			status:  http.StatusConflict,
			wantErr: "409 : ",
			f: func(*api.PortalDocument) error {
				return nil
			},
		},
		{
			name: "fail: error from func",
			doc: &api.PortalDocument{
				ID: "id",
			},
			f: func(*api.PortalDocument) error {
				return fmt.Errorf("error from f")
			},
			wantErr: "error from f",
		},
		{
			name: "pass: Get document",
			doc: &api.PortalDocument{
				ID: "id",
			},
			f: func(*api.PortalDocument) error {
				return nil
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Portal/docs/"+tt.doc.ID {
				if tt.status > 0 {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, tt.doc)
				}
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestPortal(ctx, log, ts, t)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new Portal: %s\n", t.Name(), err.Error())
		}

		// Cover without separate test
		c.NewUUID()

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Patch(ctx, tt.doc.ID, tt.f)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func newTestPortal(ctx context.Context, log *logrus.Entry, ts *httptest.Server, t *testing.T) (Portal, error) {
	host := strings.SplitAfter(ts.URL, "//")
	dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
	return NewPortal(ctx, true, dbc)
}
