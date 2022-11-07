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

func TestNewOpenShiftClusters(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	for _, tt := range []struct {
		name    string
		wantErr string
		dbName  string
		status  int
	}{
		{
			name:    "fail: DATABASE_NAME is unset",
			wantErr: "environment variable \"DATABASE_NAME\" unset (development mode)",
		},
		{
			name:    "fail: status not found",
			dbName:  "testdb",
			status:  http.StatusNotFound,
			wantErr: "404 : ",
		},
		{
			name:   "pass: Create new OpenShiftClusters",
			dbName: "testdb",
			status: http.StatusCreated,
		},
	} {
		if tt.dbName != "" {
			t.Setenv("DATABASE_NAME", tt.dbName)
		}

		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/dbs/testdb/colls/OpenShiftClusters/triggers" {
				w.WriteHeader(tt.status)
			} else {
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestOpenShiftClusters(ctx, log, ts)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})

		os.Unsetenv("DATABASE_NAME")
	}
}

func TestOpenShiftClustersCreate(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		doc     *api.OpenShiftClusterDocument
	}{
		{
			name: "fail: key isn't lowercase",
			doc: &api.OpenShiftClusterDocument{
				Key: "/SUBSCRIPTIONS/test/resourcegroups/test1/providers/my/test/partition",
			},
			status:  http.StatusCreated,
			wantErr: "key \"/SUBSCRIPTIONS/test/resourcegroups/test1/providers/my/test/partition\" is not lower case",
		},
		{
			name: "fail: status conflict",
			doc: &api.OpenShiftClusterDocument{
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			status:  http.StatusConflict,
			wantErr: "412 : ",
		},
		{
			name: "fail: partition key invalid",
			doc: &api.OpenShiftClusterDocument{
				Key: "invalid/key",
			},
			status:  http.StatusCreated,
			wantErr: "parsing failed for invalid/key. Invalid resource Id format",
		},
		{
			name: "pass: Create new OpenShiftClusters",
			doc: &api.OpenShiftClusterDocument{
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			status: http.StatusCreated,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				w.WriteHeader(tt.status)
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
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

func TestOpenShiftClustersGetPatch(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		doc     *api.OpenShiftClusterDocument
		f       func(*api.OpenShiftClusterDocument) error
		newDoc  *api.OpenShiftClusterDocument
		newDocs *api.OpenShiftClusterDocuments
	}{
		// Cover Get
		{
			name: "fail: key isn't lowercase",
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "/SUBSCRIPTIONS/test/resourcegroups/test1/providers/my/test/partition",
			},
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID:  "id",
						Key: "/SUBSCRIPTIONS/test/resourcegroups/test1/providers/my/test/partition",
					},
				},
			},
			wantErr: "key \"/SUBSCRIPTIONS/test/resourcegroups/test1/providers/my/test/partition\" is not lower case",
		},
		{
			name: "fail: invalid partition key",
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "invalid/key",
			},
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID:  "id",
						Key: "invalid/key",
					},
				},
			},
			wantErr: "parsing failed for invalid/key. Invalid resource Id format",
		},
		{
			name: "fail: No OpenShiftClusterDocuments found",
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{},
			},
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			f: func(*api.OpenShiftClusterDocument) error {
				return nil
			},
			status:  http.StatusOK,
			wantErr: "404 : ",
		},
		{
			name: "fail: More than 1 document received",
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID: "id-1",
					},
					{
						ID: "id-2",
					},
				},
			},
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			f: func(*api.OpenShiftClusterDocument) error {
				return nil
			},
			status:  http.StatusOK,
			wantErr: "read 2 documents, expected <= 1",
		},
		{

			name: "fail: 204 from raw document iterator",
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID: "id-1",
					},
					{
						ID: "id-2",
					},
				},
			},
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			f: func(*api.OpenShiftClusterDocument) error {
				return nil
			},
			status:  http.StatusNoContent,
			wantErr: "204 : ",
		},
		// Cover patch
		{
			name: "fail: error from func",
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID:  "id",
						Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
					},
				},
			},
			f: func(*api.OpenShiftClusterDocument) error {
				return fmt.Errorf("error from f")
			},
			status:  http.StatusOK,
			wantErr: "error from f",
		},
		{
			name: "fail: new document key isn't lowercase",
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID:  "id",
						Key: "/SUBSCRIPTIONS/test/resourcegroups/test1/providers/my/test/partition",
					},
				},
			},
			newDoc: &api.OpenShiftClusterDocument{},
			f: func(*api.OpenShiftClusterDocument) error {
				return nil
			},
			status:  http.StatusOK,
			wantErr: "key \"/SUBSCRIPTIONS/test/resourcegroups/test1/providers/my/test/partition\" is not lower case",
		},
		{
			name: "pass: Patch document",
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
			},
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID:  "id",
						Key: "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
					},
				},
			},
			f: func(*api.OpenShiftClusterDocument) error {
				return nil
			},
			status: http.StatusOK,
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				if tt.status == http.StatusNoContent {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, tt.newDocs)
				}
			case "/dbs/testdb/colls/OpenShiftClusters/docs/":
				encodeTestDoc(w, r, t, tt.newDocs)
			case "/dbs/testdb/colls/OpenShiftClusters/docs/" + tt.doc.ID:
				w.WriteHeader(tt.status)
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Patch(ctx, tt.doc.Key, tt.f)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenShiftClustersDequeue(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		rStatus int
		ID      string
		newDocs *api.OpenShiftClusterDocuments
		f       func(*api.OpenShiftClusterDocument) error
	}{
		{
			name:    "fail: status conflict",
			ID:      "id",
			status:  http.StatusConflict,
			wantErr: "409 : ",
		},
		{
			name:    "fail: Documents are nil",
			ID:      "id",
			newDocs: &api.OpenShiftClusterDocuments{},
		},
		{
			name: "fail: Status precondition failed",
			ID:   "id",
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID:   "id",
						Key:  "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
						ETag: "tag",
					},
				},
			},
			rStatus: http.StatusPreconditionFailed,
		},
		{
			name: "pass: Dequeue",
			ID:   "id",
			newDocs: &api.OpenShiftClusterDocuments{
				OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
					{
						ID:   "id",
						Key:  "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
						ETag: "tag",
					},
				},
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				if tt.status > 0 {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, tt.newDocs)
				}
			case "/dbs/testdb/colls/OpenShiftClusters/docs/" + tt.ID:
				if tt.rStatus > 0 {
					w.WriteHeader(tt.rStatus)
				}
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Dequeue(ctx)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

// TODO Sent document with the same uuid as c with Dequeue for Patch with lease test

func TestOpenShiftClustersPatchWithLease(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftClusterDocument
	}{
		{
			name: "pass: Patch with lease",
			doc: &api.OpenShiftClusterDocument{
				ID:   "id",
				Key:  "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
				ETag: "tag",
			},
		},
		{
			name: "fail: lost lease",
			doc: &api.OpenShiftClusterDocument{
				ID:         "id",
				Key:        "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
				LeaseOwner: "different-uuid",
				ETag:       "tag",
			},
			wantErr: "lost lease",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				// Send documents to Dequeue
				encodeTestDoc(w, r, t, &api.OpenShiftClusterDocuments{
					OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
						tt.doc,
					},
				})
			case "/dbs/testdb/colls/OpenShiftClusters/docs/" + tt.doc.ID:
				if r.Method == "GET" {
					encodeTestDoc(w, r, t, tt.doc)
				} else {
					newDoc := &api.OpenShiftClusterDocument{}
					decodeTestDoc(r, t, newDoc)
					if tt.wantErr == "lost lease" {
						newDoc.LeaseOwner = "different-uuid"
					}
					encodeTestDoc(w, r, t, newDoc)
				}
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: \n%s\n", t.Name(), err.Error())
		}

		tt.doc, err = c.Dequeue(ctx)
		if err != nil {
			t.Fatalf("\n%s\nfailed to get document with matching uuid: \n%s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.PatchWithLease(ctx, tt.doc.Key, func(*api.OpenShiftClusterDocument) error {
				return nil
			})
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenShiftClustersQueueLength(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	type data struct {
		api.MissingFields
		Document []int `json:"Documents,omitempty"`
	}

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		pr      *cosmosdb.PartitionKeyRanges
		data    *data
	}{
		{
			name:    "fail: PartitionKeyRanges not found",
			status:  http.StatusNotFound,
			wantErr: "404 : ",
		},
		{
			name: "fail: Next raw failure",
			pr: &cosmosdb.PartitionKeyRanges{
				Count: 2,
				PartitionKeyRanges: []cosmosdb.PartitionKeyRange{
					{
						ID: "1",
					},
					{
						ID: "2",
					},
				},
			},
			status:  http.StatusPreconditionFailed,
			wantErr: "412 : ",
		},
		{
			name: "pass: Get queue length",
			pr: &cosmosdb.PartitionKeyRanges{
				Count: 2,
				PartitionKeyRanges: []cosmosdb.PartitionKeyRange{
					{
						ID: "1",
					},
					{
						ID: "2",
					},
				},
			},
			data: &data{
				api.MissingFields{},
				[]int{
					1,
					2,
				},
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Resource requested: %s", r.URL.String())
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/pkranges":
				if tt.status > 0 && tt.pr == nil {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, tt.pr)
				}
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				if tt.status > 0 {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, tt.data)
				}
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.QueueLength(ctx, "OpenShiftClusters")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenShiftClustersListByPrefix(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		status  int
		subID   string
		prefix  string
		cont    string
	}{
		{
			name:    "fail: Prefix isn't lowercase",
			prefix:  "UPPER",
			wantErr: "prefix \"UPPER\" is not lower case",
		},
		{
			name:   "pass: ListByPrefix",
			prefix: "prefix",
			subID:  "subid",
			cont:   "continue",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				w.WriteHeader(http.StatusOK)
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
		}

		// Cover without separate tests
		c.ListAll(ctx)
		c.List(tt.cont)
		c.ChangeFeed()
		c.Update(ctx, &api.OpenShiftClusterDocument{})
		c.NewUUID()
		c.Lease(ctx, "key")

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.ListByPrefix(tt.subID, tt.prefix, tt.cont)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenShiftClustersDelete(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftClusterDocument
	}{
		{
			name: "fail: Prefix isn't lowercase",
			doc: &api.OpenShiftClusterDocument{
				ID:  "id",
				Key: "UPPER",
			},
			wantErr: "key \"UPPER\" is not lower case",
		},
		{
			name: "pass: Delete document",
			doc: &api.OpenShiftClusterDocument{
				ID: "id",
			},
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs/" + tt.doc.ID:
				w.WriteHeader(http.StatusNoContent)
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
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

func TestOpenShiftClustersGetByClientID(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftClusterDocument
		status  int
	}{
		{
			name: "pass: GetByClientID",
		},
		{
			name:    "fail: status precondition failed",
			status:  http.StatusPreconditionFailed,
			wantErr: "412 : ",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				if tt.status > 0 {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, &api.OpenShiftClusterDocuments{})
				}
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.GetByClientID(ctx, "part1", "client-id")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenShiftClustersEndLease(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	doc := &api.OpenShiftClusterDocument{
		ID:               "id",
		OpenShiftCluster: &api.OpenShiftCluster{},
		Key:              "/subscriptions/test/resourcegroups/test1/providers/my/test/partition",
		ETag:             "tag",
	}

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftClusterDocument
	}{
		{
			name: "pass: End lease",
			doc:  doc,
		},
		{
			name: "fail: Provisioning state failed",
			doc:  succeedProvisioningState(doc),
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				encodeTestDoc(w, r, t, &api.OpenShiftClusterDocuments{
					OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
						tt.doc,
					},
				})
			case "/dbs/testdb/colls/OpenShiftClusters/docs/" + tt.doc.ID:
				if r.Method == "GET" {
					encodeTestDoc(w, r, t, tt.doc)
				} else {
					newDoc := &api.OpenShiftClusterDocument{}
					decodeTestDoc(r, t, newDoc)
					encodeTestDoc(w, r, t, newDoc)
				}
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
		}

		provisioningState := tt.doc.OpenShiftCluster.Properties.LastProvisioningState
		failedProvisioningState := tt.doc.OpenShiftCluster.Properties.FailedProvisioningState
		s := ""

		// Satisfy patchWithLease by getting Document with LeaseOwner = c.uuid
		tt.doc, err = c.Dequeue(ctx)
		if err != nil {
			t.Fatalf("\n%s\nfailed to get document with matching uuid: \n%s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.EndLease(ctx, tt.doc.Key, provisioningState, failedProvisioningState, &s)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenShiftClustersGetByClusterResourceID(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	t.Setenv("DATABASE_NAME", "testdb")

	for _, tt := range []struct {
		name    string
		wantErr string
		doc     *api.OpenShiftClusterDocument
		status  int
	}{
		{
			name: "pass: Get by cluster resource ID",
		},
		{
			name:    "fail: 404 not found",
			status:  http.StatusNotFound,
			wantErr: "404 : ",
		},
	} {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch u := r.URL.String(); u {
			case "/dbs/testdb/colls/OpenShiftClusters/triggers":
				w.WriteHeader(http.StatusCreated)
			case "/dbs/testdb/colls/OpenShiftClusters/docs":
				if tt.status > 0 {
					w.WriteHeader(tt.status)
				} else {
					encodeTestDoc(w, r, t, &api.OpenShiftClusterDocuments{})
				}
			default:
				t.Logf("Resource not found: %s", r.URL.String())
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		c, err := newTestOpenShiftClusters(ctx, log, ts)
		if err != nil {
			t.Fatalf("\n%s\n failed to create new monitors: %s\n", t.Name(), err.Error())
		}

		t.Run(tt.name, func(t *testing.T) {
			_, err := c.GetByClusterResourceGroupID(ctx, "part1", "group1")
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Errorf("\n%v\n !=\n%v", err, tt.wantErr)
			}
		})
	}
}

func succeedProvisioningState(doc *api.OpenShiftClusterDocument) *api.OpenShiftClusterDocument {
	doc.OpenShiftCluster.Properties.LastProvisioningState = api.ProvisioningStateSucceeded
	return doc
}

func newTestOpenShiftClusters(ctx context.Context, log *logrus.Entry, ts *httptest.Server) (OpenShiftClusters, error) {
	host := strings.SplitAfter(ts.URL, "//")
	dbc := cosmosdb.NewDatabaseClient(log, ts.Client(), &codec.JsonHandle{}, host[1], nil)
	return NewOpenShiftClusters(ctx, true, dbc)
}
