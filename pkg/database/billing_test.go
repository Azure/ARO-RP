package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func TestUpdateLastBillingTimestamp(t *testing.T) {
	const original = `{"id":"billing-id","_etag":"version-1","key":"cluster-key","clusterResourceGroupIDKey":"managed-rg","infraId":"infra-id","futureField":"preserved","billing":{"creationTime":100,"deletionTime":300,"lastBillingTime":150,"location":"eastus","tenantID":"tenant","futureBillingField":"preserved"}}`
	const persisted = `{"id":"billing-id","_etag":"version-2","key":"cluster-key","clusterResourceGroupIDKey":"managed-rg","infraId":"infra-id","futureField":"preserved","billing":{"creationTime":100,"deletionTime":300,"lastBillingTime":200,"location":"eastus","tenantID":"tenant","futureBillingField":"preserved"}}`
	for _, tt := range []struct {
		name         string
		expectedETag string
		getStatus    int
		getBody      string
		putStatus    int
		putBody      string
		wantStatus   int
		wantError    string
		wantReads    int32
		wantWrites   int32
		wantDocument string
	}{
		{
			name: "missing ETag", wantError: cosmosdb.ErrETagRequired.Error(),
		},
		{
			name: "wildcard is not a version", expectedETag: "*",
			wantError: "billing document update requires a specific ETag",
		},
		{
			name: "document not found", expectedETag: "version-1",
			getStatus: http.StatusNotFound, getBody: `{"code":"NotFound"}`,
			wantStatus: http.StatusNotFound, wantReads: 1,
		},
		{
			name: "read failure", expectedETag: "version-1",
			getStatus: http.StatusInternalServerError, getBody: `{"code":"InternalServerError"}`,
			wantStatus: http.StatusInternalServerError, wantReads: 1,
		},
		{
			name: "missing billing payload", expectedETag: "version-1",
			getStatus: http.StatusOK, getBody: `{"id":"billing-id","_etag":"version-1"}`,
			wantError: `billing document "billing-id" is missing billing data`, wantReads: 1,
		},
		{
			name: "stale decision rejected before write", expectedETag: "older-version",
			getStatus: http.StatusOK, getBody: original,
			wantStatus: http.StatusPreconditionFailed, wantReads: 1,
		},
		{
			name: "concurrent write rejected without retry", expectedETag: "version-1",
			getStatus: http.StatusOK, getBody: original,
			putStatus: http.StatusPreconditionFailed, putBody: `{"code":"PreconditionFailed","message":"concurrent writer"}`,
			wantStatus: http.StatusPreconditionFailed, wantReads: 1, wantWrites: 1,
		},
		{
			name: "replacement failure", expectedETag: "version-1",
			getStatus: http.StatusOK, getBody: original,
			putStatus: http.StatusInternalServerError, putBody: `{"code":"InternalServerError"}`,
			wantStatus: http.StatusInternalServerError, wantReads: 1, wantWrites: 1,
		},
		{
			name: "update preserves fields and returns persisted ETag", expectedETag: "version-1",
			getStatus: http.StatusOK, getBody: original,
			putStatus: http.StatusOK, putBody: persisted,
			wantReads: 1, wantWrites: 1, wantDocument: persisted,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var reads, writes atomic.Int32
			h := &codec.JsonHandle{}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/dbs/test-db/colls/Billing/docs/billing-id" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if got := r.Header.Get("X-Ms-Documentdb-Partitionkey"); got != `["billing-id"]` {
					t.Errorf("partition key: got %q", got)
				}
				switch r.Method {
				case http.MethodGet:
					reads.Add(1)
					if tt.getStatus == 0 {
						t.Error("unexpected read")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(tt.getStatus)
					if _, err := io.WriteString(w, tt.getBody); err != nil {
						t.Error(err)
					}
				case http.MethodPut:
					writes.Add(1)
					if got := r.Header.Get("If-Match"); got != "version-1" {
						t.Errorf("If-Match: want version-1, got %q", got)
					}
					var got, want api.BillingDocument
					if err := codec.NewDecoder(r.Body, h).Decode(&got); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if err := codec.NewDecoder(strings.NewReader(strings.Replace(persisted, "version-2", "version-1", 1)), h).Decode(&want); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					if diff := cmp.Diff(want, got, cmp.AllowUnexported(api.MissingFields{})); diff != "" {
						t.Errorf("replacement (-want +got):\n%s", diff)
					}
					if tt.putStatus == 0 {
						t.Error("unexpected replacement")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(tt.putStatus)
					if _, err := io.WriteString(w, tt.putBody); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected method: %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()

			dbc := cosmosdb.NewDatabaseClient(nil, server.Client(), h, strings.TrimPrefix(server.URL, "https://"), nil)
			client, err := NewBilling(context.Background(), dbc, "test-db")
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.UpdateLastBillingTimestamp(context.Background(), "billing-id", tt.expectedETag, 200)
			switch {
			case tt.wantStatus != 0:
				if !cosmosdb.IsErrorStatusCode(err, tt.wantStatus) {
					t.Fatalf("want status %d, got %v", tt.wantStatus, err)
				}
			case tt.wantError != "":
				if err == nil || err.Error() != tt.wantError {
					t.Fatalf("want error %q, got %v", tt.wantError, err)
				}
				if tt.expectedETag == "" && !errors.Is(err, cosmosdb.ErrETagRequired) {
					t.Errorf("missing ETag error identity lost: %v", err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
			var want *api.BillingDocument
			if tt.wantDocument != "" {
				if err := codec.NewDecoder(strings.NewReader(tt.wantDocument), h).Decode(&want); err != nil {
					t.Fatal(err)
				}
			}
			if diff := cmp.Diff(want, got, cmp.AllowUnexported(api.MissingFields{})); diff != "" {
				t.Errorf("result (-want +got):\n%s", diff)
			}
			if reads.Load() != tt.wantReads || writes.Load() != tt.wantWrites {
				t.Errorf("requests: want %d reads/%d writes, got %d reads/%d writes", tt.wantReads, tt.wantWrites, reads.Load(), writes.Load())
			}
		})
	}
}
