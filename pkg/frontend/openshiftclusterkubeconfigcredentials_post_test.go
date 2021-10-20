package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/v20210901preview"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestPostOpenShiftClusterKubeConfigCredentials(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"2021-09-01-preview": api.APIs["2021-09-01-preview"],
		"no-credentials": {
			OpenShiftClusterConverter:       api.APIs["2021-09-01-preview"].OpenShiftClusterConverter,
			OpenShiftClusterStaticValidator: api.APIs["2021-09-01-preview"].OpenShiftClusterStaticValidator,
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"
	resourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID)

	type test struct {
		name           string
		resourceID     string
		apiVersion     string
		fixture        func(*testdatabase.Fixture)
		dbError        error
		wantStatusCode int
		wantResponse   func(*test) *v20210901preview.OpenShiftClusterAdminKubeconfig
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "cluster exists in db",
			resourceID: resourceID,
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateSucceeded,
							UserAdminKubeconfig: api.SecureBytes("Y2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBDZXJ0RGF0YVRvQmVSZW1vdmVkCiAgICBzZXJ2ZXI6IGh0dHBzOi8vYXBpLWludC5yYW5kb21pZC5yZWdpb24uYXJvYXBwLmlvOjY0NDMKICBuYW1lOiByYW5kb21pZApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogcmFuZG9taWQKICAgIHVzZXI6IHN5c3RlbTphZG1pbgogIG5hbWU6IHN5c3RlbTphZG1pbgpjdXJyZW50LWNvbnRleHQ6IHN5c3RlbTphZG1pbgpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogc3lzdGVtOmFkbWluCiAgdXNlcjoKICAgIGNsaWVudC1jZXJ0aWZpY2F0ZS1kYXRhOiBDbGllbnRDZXJ0CiAgICBjbGllbnQta2V5LWRhdGE6IENsaWVudEtleQo="),
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagAdminKubeconfig,
									State: "Registered",
								},
							},
						},
					},
				})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20210901preview.OpenShiftClusterAdminKubeconfig {
				kubeconfig, err := base64.StdEncoding.DecodeString("Y2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIHNlcnZlcjogaHR0cHM6Ly9hcGkucmFuZG9taWQucmVnaW9uLmFyb2FwcC5pbzo2NDQzCiAgbmFtZTogcmFuZG9taWQKY29udGV4dHM6Ci0gY29udGV4dDoKICAgIGNsdXN0ZXI6IHJhbmRvbWlkCiAgICB1c2VyOiBzeXN0ZW06YWRtaW4KICBuYW1lOiBzeXN0ZW06YWRtaW4KY3VycmVudC1jb250ZXh0OiBzeXN0ZW06YWRtaW4KcHJlZmVyZW5jZXM6IHt9CnVzZXJzOgotIG5hbWU6IHN5c3RlbTphZG1pbgogIHVzZXI6CiAgICBjbGllbnQtY2VydGlmaWNhdGUtZGF0YTogQ2xpZW50Q2VydAogICAgY2xpZW50LWtleS1kYXRhOiBDbGllbnRLZXkK")
				if err != nil {
					panic(err)
				}
				return &v20210901preview.OpenShiftClusterAdminKubeconfig{
					Kubeconfig: kubeconfig,
				}
			},
		},
		{
			name:       "cluster exists in db but no feature flag",
			resourceID: resourceID,
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateSucceeded,
							UserAdminKubeconfig: api.SecureBytes("{kubeconfig}"),
						},
					},
				})
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
			wantStatusCode: http.StatusForbidden,
			wantError:      `403: Forbidden: : Subscription feature flag 'Microsoft.RedHatOpenShift/AdminKubeconfig' is not enabled on this subscription to use this API.`,
		},
		{
			name:           "credentials request is not allowed in the API version",
			resourceID:     resourceID,
			apiVersion:     "no-credentials",
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidResourceType: : The resource type 'openshiftclusters' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'no-credentials'.`,
		},
		{
			name:       "cluster exists in db in creating state",
			resourceID: resourceID,
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateCreating,
							UserAdminKubeconfig: api.SecureBytes("{kubeconfig}"),
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagAdminKubeconfig,
									State: "Registered",
								},
							},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Creating'.`,
		},
		{
			name:       "cluster exists in db in deleting state",
			resourceID: resourceID,
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:   api.ProvisioningStateDeleting,
							UserAdminKubeconfig: api.SecureBytes("{kubeconfig}"),
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagAdminKubeconfig,
									State: "Registered",
								},
							},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Deleting'.`,
		},
		{
			name:       "cluster failed to create",
			resourceID: resourceID,
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							UserAdminKubeconfig:     api.SecureBytes("{}"),
							FailedProvisioningState: api.ProvisioningStateCreating,
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagAdminKubeconfig,
									State: "Registered",
								},
							},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Failed'.`,
		},
		{
			name:       "cluster failed to delete",
			resourceID: resourceID,
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateDeleting,
							UserAdminKubeconfig:     api.SecureBytes("{kubeconfig}"),
						},
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagAdminKubeconfig,
									State: "Registered",
								},
							},
						},
					},
				})
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Failed'.`,
		},
		{
			name:       "cluster not found in db",
			resourceID: resourceID,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
							RegisteredFeatures: []api.RegisteredFeatureProfile{
								{
									Name:  api.FeatureFlagAdminKubeconfig,
									State: "Registered",
								},
							},
						},
					},
				})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:           "internal error",
			resourceID:     resourceID,
			dbError:        &cosmosdb.Error{Code: "500", Message: "oh no!"},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			if tt.dbError != nil {
				ti.subscriptionsClient.SetError(tt.dbError)
				ti.openShiftClustersClient.SetError(tt.dbError)
			}

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, apis, &noop.Noop{}, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			reqAPIVersion := "2021-09-01-preview"
			if tt.apiVersion != "" {
				reqAPIVersion = tt.apiVersion
			}

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server%s/listAdminCredentials?api-version=%s", tt.resourceID, reqAPIVersion),
				nil, nil)
			if err != nil {
				t.Error(err)
			}

			var wantResponse interface{}
			if tt.wantResponse != nil {
				wantResponse = tt.wantResponse(tt)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
