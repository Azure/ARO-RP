package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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
	ctx := t.Context()
	type test struct {
		name           string
		billingDocId   string
		fixture        func(f *testdatabase.Fixture)
		compareOption  cmp.Option
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
			compareOption:  cmpopts.IgnoreFields(admin.Billing{}, "CreationTime"),
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
			compareOption:  cmpopts.IgnoreFields(admin.Billing{}, "CreationTime"),
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
				"https://server/admin/billingDocuments/"+tt.billingDocId,
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse, tt.compareOption)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
