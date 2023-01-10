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

func TestNewOpenshiftVersions(t *testing.T) {
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
			name:   "pass: Create new OpenshiftVersions",
			dbName: "testdb",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", tt.dbName)
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestOpenshiftVersions(ctx, log, ts)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		os.Unsetenv("DATABASE_NAME")
	}
}

func TestOpenshiftVersionsCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftVersionDocument
	}{
		{
			name: "fail: ID isn't lowercase",
			doc: &api.OpenShiftVersionDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Create document",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/OpenShiftVersions/docs" {
				w.WriteHeader(http.StatusCreated)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenshiftVersions(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new OpenshiftVersions: %s\n", t.Name(), err.Error())
		}

		// Cover without separate test
		c.ChangeFeed()
		c.NewUUID()
		c.ListAll(ctx)

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Create(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenshiftVersionsListAll(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftVersionDocument
	}{
		{
			name: "pass: Create document",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/OpenShiftVersions/docs" {
				w.WriteHeader(http.StatusCreated)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenshiftVersions(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new OpenshiftVersions: %s\n", t.Name(), err.Error())
		}

		// Cover without sep.rate test
		c.ChangeFeed()
		c.NewUUID()

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Create(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenshiftVersionsDelete(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftVersionDocument
	}{
		{
			name: "ID isn't lowercase",
			doc: &api.OpenShiftVersionDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Delete document",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/OpenShiftVersions/docs/"+tt.doc.ID {
				w.WriteHeader(http.StatusNoContent)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenshiftVersions(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new OpenshiftVersions: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			err := c.Delete(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenshiftVersionsPatch(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftVersionDocument
		f       func(*api.OpenShiftVersionDocument) error
		status  int
	}{
		{
			// Covers Get
			name: "ID isn't lowercase",
			doc: &api.OpenShiftVersionDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
			status:  http.StatusOK,
			f: func(*api.OpenShiftVersionDocument) error {
				return nil
			},
		},
		{
			name: "fail: status not found",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
			status:  http.StatusNotFound,
			wantErr: "404 : ",
			f: func(*api.OpenShiftVersionDocument) error {
				return nil
			},
		},
		{
			name: "fail: error from func",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
			f: func(*api.OpenShiftVersionDocument) error {
				return fmt.Errorf("error from f")
			},
			wantErr: "error from f",
		},
		{
			name: "pass: Get document",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
			f: func(*api.OpenShiftVersionDocument) error {
				return nil
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/OpenShiftVersions/docs/"+tt.doc.ID {
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

		c, err := newTestOpenshiftVersions(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new OpenshiftVersions: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Patch(ctx, tt.doc.ID, tt.f)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenshiftVersionsUpdate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftVersionDocument
		f       func(*api.OpenShiftVersionDocument) error
		status  int
	}{
		{
			name: "ID isn't lowercase",
			doc: &api.OpenShiftVersionDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "fail: status not found",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
			wantErr: "404 : ",
			status:  http.StatusNotFound,
		},
		{
			name: "pass: Update document",
			doc: &api.OpenShiftVersionDocument{
				ID: "id",
			},
			f: func(*api.OpenShiftVersionDocument) error {
				return nil
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/OpenShiftVersions/docs/"+tt.doc.ID {
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

		c, err := newTestOpenshiftVersions(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new OpenshiftVersions: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Update(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func newTestOpenshiftVersions(ctx context.Context, log *logrus.Entry, ts *httptest.Server) (OpenShiftVersions, error) {
	host := strings.SplitAfter(ts.URL, "//")
	dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
	return NewOpenShiftVersions(ctx, true, dbc)
}
