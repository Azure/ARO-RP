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
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestAdminListBillingDocuments(t *testing.T) {
	ctx := t.Context()

	type test struct {
		name           string
		throwsError    error
		fixture        func(*testdatabase.Fixture)
		compareOption  cmp.Option
		wantStatusCode int
		wantResponse   *admin.BillingDocumentList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "billing documents exist in db",
			fixture: func(f *testdatabase.Fixture) {
				f.AddBillingDocuments(
					&api.BillingDocument{
						ID:                        "00000000-0000-0000-0000-000000000001",
						Key:                       "key1",
						ClusterResourceGroupIDKey: "cluster-rg-key-1",
						InfraID:                   "infra-1",
						Billing: &api.Billing{
							DeletionTime:    0,
							LastBillingTime: 1500,
							Location:        "eastus",
							TenantID:        "tenant-1",
						},
					},
					&api.BillingDocument{
						ID:                        "00000000-0000-0000-0000-000000000002",
						Key:                       "key2",
						ClusterResourceGroupIDKey: "cluster-rg-key-2",
						InfraID:                   "infra-2",
						Billing: &api.Billing{
							DeletionTime:    0,
							LastBillingTime: 2500,
							Location:        "westus",
							TenantID:        "tenant-2",
						},
					})
			},
			compareOption:  cmpopts.IgnoreFields(admin.Billing{}, "CreationTime"),
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.BillingDocumentList{
				BillingDocuments: []*admin.BillingDocument{
					{
						ID:                        "00000000-0000-0000-0000-000000000001",
						Key:                       "key1",
						ClusterResourceGroupIDKey: "cluster-rg-key-1",
						InfraID:                   "infra-1",
						Billing: &admin.Billing{
							LastBillingTime: 1500,
							Location:        "eastus",
							TenantID:        "tenant-1",
						},
					},
					{
						ID:                        "00000000-0000-0000-0000-000000000002",
						Key:                       "key2",
						ClusterResourceGroupIDKey: "cluster-rg-key-2",
						InfraID:                   "infra-2",
						Billing: &admin.Billing{
							LastBillingTime: 2500,
							Location:        "westus",
							TenantID:        "tenant-2",
						},
					},
				},
			},
		},
		{
			name:           "no billing documents found in db",
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.BillingDocumentList{
				BillingDocuments: []*admin.BillingDocument{},
			},
		},
		{
			name:           "internal error while iterating list",
			wantStatusCode: http.StatusInternalServerError,
			throwsError:    &cosmosdb.Error{StatusCode: 500, Code: "ERR500", Message: "random error"},
			wantError:      `500: InternalServerError: : 500 ERR500: random error`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithBilling()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			aead := testdatabase.NewFakeAEAD()

			if tt.throwsError != nil {
				ti.billingClient.SetError(tt.throwsError)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, aead, nil, nil, nil, nil, nil, ti.enricher)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/billingDocuments",
				http.Header{
					"Referer": []string{"https://mockrefererhost/"},
				}, nil)
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
