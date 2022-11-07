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

func TestNewMonitors(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	for _, tt := range []struct {
		name    string
		wantErr string
		dbName  string
		status  int
	}{
		{
			name:   "pass: create new monitors",
			status: http.StatusCreated,
			dbName: "testdb",
		},
		{
			name:    "fail: DATABASE_NAME is unset",
			wantErr: "environment variable \"DATABASE_NAME\" unset (development mode)",
		},
		{
			name:    "fail: http status conflict",
			wantErr: "404 : ",
			dbName:  "testdb",
			status:  http.StatusNotFound,
		},
	} {
		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", tt.dbName)
		}

		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Monitors/triggers" {
				w.WriteHeader(tt.status)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
			}
		}))

		host := strings.SplitAfter(ts.URL, "//")

		dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)

		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMonitors(ctx, true, dbc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		os.Unsetenv("DATABASE_NAME")
	}
}

func TestMonitorsCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		doc     *api.MonitorDocument
	}{
		{
			name: "fail: id isn't lowercase",
			doc: &api.MonitorDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
			status:  http.StatusCreated,
		},
		{
			name: "fail: http status conflict",
			doc: &api.MonitorDocument{
				ID: "id-1",
			},
			status:  http.StatusConflict,
			wantErr: "412 : ",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/Monitors/triggers" {
				w.WriteHeader(http.StatusCreated)
			} else if r.URL.String() == "/dbs/testdb/colls/Monitors/docs" {
				w.WriteHeader(tt.status)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
			}
		}))

		c := newTestMonitor(ctx, log, ts, t)

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Create(ctx, tt.doc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestMonitorsPatchWithLease(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.MonitorDocument
		f       func(*api.MonitorDocument) error
	}{
		{
			name: "fail: lost lease",
			doc: &api.MonitorDocument{
				ID:         "id-1",
				LeaseOwner: "different-uuid",
			},
			f: func(doc *api.MonitorDocument) error {
				return nil
			},
			wantErr: "lost lease",
		},
		{
			name: "fail: id isn't lowercase",
			doc: &api.MonitorDocument{
				ID: "UPPER",
			},
			wantErr: "id \"UPPER\" is not lower case",
		},
		{
			name: "pass: Patch with lease",
			f: func(doc *api.MonitorDocument) error {
				return nil
			},
			doc: &api.MonitorDocument{
				ID:   "master",
				ETag: "tag",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/Monitors/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/Monitors/docs":
				// send documents to try lease
				encodeTestDoc(w, r, t, &api.MonitorDocuments{
					MonitorDocuments: []*api.MonitorDocument{
						tt.doc,
					},
				})
			case "/dbs/testdb/colls/Monitors/docs/" + tt.doc.ID:
				if r.Method == "GET" {
					encodeTestDoc(w, r, t, tt.doc)
				} else {
					newDoc := &api.MonitorDocument{}
					decodeTestDoc(r, t, newDoc)
					encodeTestDoc(w, r, t, newDoc)
				}
			default:
				w.WriteHeader(http.StatusNotFound)
				t.Logf("Resource not found: %s", r.URL.String())
			}
		}))

		c := newTestMonitor(ctx, log, ts, t)

		// Use TryLease to get document with c's UUID as LeaseOwner
		// Tests will always fail at lost lease unless the document has the same uuid
		var err error
		if tt.doc.ID == "master" {
			tt.doc, err = c.TryLease(ctx)
			if err != nil {
				t.Fatalf("\n%s\n failed try lease: %s\n", tt.name, err.Error())
			}
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.PatchWithLease(ctx, tt.doc.ID, tt.f)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestMonitorsTryLease(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		doc     *api.MonitorDocument
	}{
		{
			name:    "fail: document not found",
			status:  http.StatusNotFound,
			wantErr: "404 : ",
		},
		{
			name: "fail: id isn't lowercase",
			doc: &api.MonitorDocument{
				ID:   "MASTER",
				ETag: "tag",
			},
			wantErr: "id \"MASTER\" is not lower case",
		},
		{
			name:   "fail: status precondition",
			status: http.StatusPreconditionFailed,
			doc: &api.MonitorDocument{
				ID:   "master",
				ETag: "tag",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/Monitors/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/Monitors/docs":
				if tt.status == http.StatusNotFound {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, &api.MonitorDocuments{
						MonitorDocuments: []*api.MonitorDocument{
							tt.doc,
						},
					})
				}
			case "/dbs/testdb/colls/Monitors/docs/" + tt.doc.ID:
				if tt.status > 0 {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, tt.doc)
				}
			default:
				t.Logf("Resource Requested not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestMonitor(ctx, log, ts, t)

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.TryLease(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestListBuckets(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name     string
		wantErr  string
		status   int
		doc      *api.MonitorDocument
		tryLease bool
		count    int
	}{
		{
			name:    "fail: 404 not found",
			status:  http.StatusNotFound,
			wantErr: "404 : ",
			doc: &api.MonitorDocument{
				ID:   "master",
				ETag: "tag",
				Monitor: &api.Monitor{
					Buckets: []string{""},
				},
			},
		},
		{
			name: "pass: range over buckets",
			doc: &api.MonitorDocument{
				ID: "master",
				Monitor: &api.Monitor{
					Buckets: []string{
						"bucket1",
					},
				},
				ETag: "tag",
			},
			tryLease: true,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/Monitors/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/Monitors/docs":
				// send documents to TryLease
				encodeTestDoc(w, r, t, &api.MonitorDocuments{
					MonitorDocuments: []*api.MonitorDocument{
						tt.doc,
					},
				})
			case "/dbs/testdb/colls/Monitors/docs/master":
				if tt.status > 0 {
					w.WriteHeader(tt.status)
					// TryLease and List Buckets use the same URI
				} else if tt.count == 0 {
					newDoc := &api.MonitorDocument{}
					decodeTestDoc(r, t, newDoc)
					tt.doc = newDoc
					encodeTestDoc(w, r, t, newDoc)
				} else {
					encodeTestDoc(w, r, t, tt.doc)
				}
			default:
				t.Logf("Resource Requested not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestMonitor(ctx, log, ts, t)

		if tt.tryLease {
			newDoc, err := c.TryLease(ctx)
			if err != nil {
				t.Fatalf("\n%s\n failed try lease: %s\n", tt.name, err.Error())
			}
			tt.doc.Monitor.Buckets[0] = newDoc.LeaseOwner
			tt.count++
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.ListBuckets(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestListMonitors(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
	}{
		{
			name:   "pass: list monitors",
			status: http.StatusOK,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/Monitors/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/Monitors/docs":
				w.WriteHeader(tt.status)
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestMonitor(ctx, log, ts, t)

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.ListMonitors(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestMonitorHeartbeat(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
	}{
		{
			name:    "fail: document not found",
			status:  http.StatusNotFound,
			wantErr: "404 : ",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/Monitors/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/Monitors/docs":
				w.WriteHeader(tt.status)
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c := newTestMonitor(ctx, log, ts, t)

		t.Run(tt.name, func(t *testing.T) {
			err := c.MonitorHeartbeat(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

// decodeTestDoc use c.uuid from TryLease request to return doc with c.uuid as LeaseOwner
func decodeTestDoc(r *http.Request, t *testing.T, doc interface{}) {
	if r.Body != http.NoBody {
		err := codec.NewDecoder(r.Body, &codec.JsonHandle{}).Decode(doc)
		if err != nil {
			t.Fatalf("\n%s\nfailed to decode document from request body\n%v\n", t.Name(), err)
		}
	}
}

func encodeTestDoc(w http.ResponseWriter, r *http.Request, t *testing.T, docs interface{}) {
	buf := &bytes.Buffer{}
	err := codec.NewEncoder(buf, &codec.JsonHandle{}).Encode(docs)
	if err != nil {
		t.Logf("\n%s\nfailed to encode document to request body\n%v\n", t.Name(), err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-ms-version", "2018-12-31")
	w.Write(buf.Bytes())
}

func newTestMonitor(ctx context.Context, log *logrus.Entry, ts *httptest.Server, t *testing.T) Monitors {
	host := strings.SplitAfter(ts.URL, "//")
	dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
	c, err := NewMonitors(ctx, true, dbc)
	if err != nil {
		t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
	}
	return c
}
