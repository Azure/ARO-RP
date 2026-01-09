package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminGetBillingDocument(t *testing.T) {
	type test struct {
		name           string
		billingDocId   string
		fixture        func(f *testdatabase.Fixture)
		wantStatusCode int
		wantError      string
		wantResponse   *admin.BillingDocument
	}

	for _, tt := range []*test{
		{
			name:           "no billing document found",
			billingDocId:   "non-existent-id",
			wantError:      "404: NotFound: : billing document not found: 404 : ",
			fixture:        func(f *testdatabase.Fixture) {},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:         "get billing document",
			billingDocId: "00000000-0000-0000-0000-000000000001",
			fixture: func(f *testdatabase.Fixture) {
				f.AddBillingDocuments(&api.BillingDocument{
					ID:                        "00000000-0000-0000-0000-000000000001",
					Key:                       "test-key",
					ClusterResourceGroupIDKey: "test-cluster-rg-key",
					InfraID:                   "test-infra-id",
					Billing: &api.Billing{
						DeletionTime:    0,
						LastBillingTime: 1500,
						Location:        "eastus",
						TenantID:        "test-tenant-id",
					},
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.BillingDocument{
				ID:                        "00000000-0000-0000-0000-000000000001",
				Key:                       "test-key",
				ClusterResourceGroupIDKey: "test-cluster-rg-key",
				InfraID:                   "test-infra-id",
				Billing: &admin.Billing{
					LastBillingTime: 1500,
					Location:        "eastus",
					TenantID:        "test-tenant-id",
				},
			},
		},
		{
			name:         "get billing document with deletion time",
			billingDocId: "00000000-0000-0000-0000-000000000002",
			fixture: func(f *testdatabase.Fixture) {
				f.AddBillingDocuments(&api.BillingDocument{
					ID:                        "00000000-0000-0000-0000-000000000002",
					Key:                       "deleted-key",
					ClusterResourceGroupIDKey: "deleted-cluster-rg-key",
					InfraID:                   "deleted-infra-id",
					Billing: &api.Billing{
						DeletionTime:    2000,
						LastBillingTime: 1500,
						Location:        "westus",
						TenantID:        "deleted-tenant-id",
					},
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.BillingDocument{
				ID:                        "00000000-0000-0000-0000-000000000002",
				Key:                       "deleted-key",
				ClusterResourceGroupIDKey: "deleted-cluster-rg-key",
				InfraID:                   "deleted-infra-id",
				Billing: &admin.Billing{
					DeletionTime:    2000,
					LastBillingTime: 1500,
					Location:        "westus",
					TenantID:        "deleted-tenant-id",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ti := newTestInfra(t).WithBilling()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/providers/Microsoft.RedHatOpenShift/billingDocuments/"+tt.billingDocId,
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantError == "" {
				// Validate status code for success cases
				if resp.StatusCode != tt.wantStatusCode {
					t.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, tt.wantStatusCode)
				}

				var doc admin.BillingDocument
				err = json.Unmarshal(b, &doc)
				if err != nil {
					t.Fatal(err)
				}

				// Verify creationTime was set by database trigger
				if doc.Billing != nil && doc.Billing.CreationTime == 0 {
					t.Error("CreationTime should be set by database trigger")
				}

				// Use cmp.Diff with IgnoreFields to ignore auto-generated CreationTime
				if diff := cmp.Diff(tt.wantResponse, &doc, cmpopts.IgnoreFields(admin.Billing{}, "CreationTime")); diff != "" {
					t.Errorf("billing document mismatch (-want +got):\n%s", diff)
				}
			} else {
				// Validate error cases using validateResponse
				err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, nil)
				if err != nil {
					t.Error(err)
				}

				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				if err != nil {
					t.Fatal(err)
				}

				if cloudErr.Error() != tt.wantError {
					t.Error(cloudErr)
				}
			}
		})
	}
}
