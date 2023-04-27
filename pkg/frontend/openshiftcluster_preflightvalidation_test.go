package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestPreflightValidation(t *testing.T) {
	ctx := context.Background()
	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name             string
		preflightRequest func() *api.PreflightRequest
		fixture          func(*testdatabase.Fixture)
		wantStatusCode   int
		wantError        string
		wantResponse     *api.ValidationResult
	}
	for _, tt := range []*test{
		{
			name: "Successful Preflight",
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				})
			},
			preflightRequest: func() *api.PreflightRequest {
				return &api.PreflightRequest{
					Resources: []json.RawMessage{
						[]byte(`
								{
									"apiVersion": "2022-04-01",
									"name": "resourceName",
									"type": "microsoft.redhatopenshift/openshiftclusters",
									"location": "eastus",
									"properties": {
										"clusterProfile": {
										  "domain": "example.aroapp.io",
										  "resourceGroupId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/resourcename",
										  "fipsValidatedModules": "Enabled"
										},
										"consoleProfile": {},
										"servicePrincipalProfile": {
										  "clientId": "00000000-0000-0000-1111-000000000000",
										  "clientSecret": "00000000-0000-0000-0000-000000000000"
										},
										"networkProfile": {
										  "podCidr": "10.128.0.0/14",
										  "serviceCidr": "172.30.0.0/16"
										},
										"masterProfile": {
										  "vmSize": "Standard_D32s_v3",
										  "subnetId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/ms-eastus/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/CARO2-master",
										  "encryptionAtHost": "Enabled",
										  "diskEncryptionSetId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/ms-eastus/providers/Microsoft.Compute/diskEncryptionSets/ms-eastus-disk-encryption-set"
										},
										"workerProfiles": [
										  {
											"name": "worker",
											"vmSize": "Standard_D32s_v3",
											"diskSizeGB": 128,
											"subnetId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/ms-eastus/providers/Microsoft.Network/virtualNetworks/dev-vnet/subnets/CARO2-worker",
											"count": 3,
											"encryptionAtHost": "Enabled",
											"diskEncryptionSetId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/ms-eastus/providers/Microsoft.Compute/diskEncryptionSets/ms-eastus-disk-encryption-set"
										  }
										],
										"apiserverProfile": {
										  "visibility": "Public"
										},
										"ingressProfiles": [
										  {
											"name": "default",
											"visibility": "Public"
										  }
										]
									  }
								}
						`),
					},
				}
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &api.ValidationResult{
				Status: api.ValidationStatusSucceeded,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			oc := tt.preflightRequest()

			go f.Run(ctx, nil, nil)

			headers := http.Header{
				"Content-Type": []string{"application/json"},
			}

			resp, b, err := ti.request(http.MethodPost,
				"https://server"+testdatabase.GetPreflightPath(mockSubID, "deploymentName")+"?api-version=2020-04-30",
				headers, oc)
			if err != nil {
				t.Error(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
