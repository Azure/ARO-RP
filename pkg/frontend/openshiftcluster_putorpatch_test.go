package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

var mockSubID = "00000000-0000-0000-0000-000000000000"
var mockCurrentTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

type dummyOpenShiftClusterValidator struct{}

func (*dummyOpenShiftClusterValidator) Static(interface{}, *api.OpenShiftCluster) error {
	return nil
}

func TestPutOrPatchOpenShiftClusterAdminAPI(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"admin": {
			OpenShiftClusterConverter: api.APIs["admin"].OpenShiftClusterConverter,
			OpenShiftClusterStaticValidator: func(string, string, bool, string) api.OpenShiftClusterStaticValidator {
				return &dummyOpenShiftClusterValidator{}
			},
			OpenShiftClusterCredentialsConverter: api.APIs["admin"].OpenShiftClusterCredentialsConverter,
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		request        func(*admin.OpenShiftCluster)
		isPatch        bool
		fixture        func(*testdatabase.Fixture)
		wantStatusCode int
		wantEnriched   []string
		wantDocuments  func(*testdatabase.Checker)
		wantResponse   *admin.OpenShiftCluster
		wantAsync      bool
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "patch with empty request",
			request: func(oc *admin.OpenShiftCluster) {
			},
			isPatch: true,
			fixture: func(f *testdatabase.Fixture) {
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
					},
				})
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
						},
					},
				})
			},
			wantEnriched: []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
						},
					},
				})
			},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftCluster{
				ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
				Type: "Microsoft.RedHatOpenShift/openShiftClusters",
				Tags: map[string]string{"tag": "will-be-kept"},
				Properties: admin.OpenShiftClusterProperties{
					ProvisioningState:     admin.ProvisioningStateAdminUpdating,
					LastProvisioningState: admin.ProvisioningStateSucceeded,
					ClusterProfile: admin.ClusterProfile{
						FipsValidatedModules: admin.FipsValidatedModulesDisabled,
					},
					NetworkProfile: admin.NetworkProfile{
						SoftwareDefinedNetwork: admin.SoftwareDefinedNetworkOpenShiftSDN,
					},
					MasterProfile: admin.MasterProfile{
						EncryptionAtHost: admin.EncryptionAtHostDisabled,
					},
				},
			},
		},
		{
			name: "patch a cluster with registry profile should ignore registry profile",
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
				oc.Name = "resourceName"
				oc.Properties.RegistryProfiles = []admin.RegistryProfile{
					{
						Name:     "TestUser",
						Username: "TestUserName",
					},
				}
			},
			isPatch: true,
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
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
						},
					},
				})
			},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateAdminUpdating,
						ProvisioningState:        api.ProvisioningStateAdminUpdating,
					},
				})
				c.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain:               "changed",
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
						},
					}})
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftCluster{
				ID:   testdatabase.GetResourcePath(mockSubID, "resourceName"),
				Name: "resourceName",
				Type: "Microsoft.RedHatOpenShift/openShiftClusters",
				Tags: map[string]string{"tag": "will-be-kept"},
				Properties: admin.OpenShiftClusterProperties{
					ProvisioningState:     admin.ProvisioningStateAdminUpdating,
					LastProvisioningState: admin.ProvisioningStateSucceeded,
					ClusterProfile: admin.ClusterProfile{
						Domain:               "changed",
						FipsValidatedModules: admin.FipsValidatedModulesDisabled,
					},
					NetworkProfile: admin.NetworkProfile{
						SoftwareDefinedNetwork: admin.SoftwareDefinedNetworkOpenShiftSDN,
					},
					MasterProfile: admin.MasterProfile{
						EncryptionAtHost: admin.EncryptionAtHostDisabled,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithAsyncOperations().
				WithSubscriptions()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, apis, &noop.Noop{}, nil, nil, nil, func(log *logrus.Entry, dialer proxy.Dialer, m metrics.Interface) clusterdata.OpenShiftClusterEnricher {
				return ti.enricher
			})
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).bucketAllocator = bucket.Fixed(1)

			var systemDataEnricherCalled bool
			f.(*frontend).systemDataEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				systemDataEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)

			oc := &admin.OpenShiftCluster{}
			if tt.request != nil {
				tt.request(oc)
			}

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			}

			resp, b, err := ti.request(method,
				"https://server"+testdatabase.GetResourcePath(mockSubID, "resourceName")+"?api-version=admin",
				http.Header{
					"Content-Type": []string{"application/json"},
				}, oc)
			if err != nil {
				t.Error(err)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}
			errs := validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			for _, err := range errs {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
			}
			errs = ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)
			for _, i := range errs {
				t.Error(i)
			}
			errs = ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
			for _, i := range errs {
				t.Error(i)
			}

			errs = ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}

			if !systemDataEnricherCalled {
				t.Error(systemDataEnricherCalled)
			}
		})
	}

}

func baseDocument() *api.OpenShiftClusterDocument {
	return &api.OpenShiftClusterDocument{
		Key:    strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
		Bucket: 1,
		OpenShiftCluster: &api.OpenShiftCluster{
			ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
			Name:     "resourceName",
			Location: "eastus",
			Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
			Properties: api.OpenShiftClusterProperties{
				ArchitectureVersion: version.InstallArchitectureVersion,
				ProvisioningState:   api.ProvisioningStateSucceeded,
				ProvisionedBy:       version.GitCommit,
				CreatedAt:           mockCurrentTime,
				CreatedBy:           version.GitCommit,
				ClusterProfile: api.ClusterProfile{
					Version:              version.InstallStream.Version.String(),
					FipsValidatedModules: api.FipsValidatedModulesDisabled,
					Domain:               "example.eastus.aroapp.io",
					ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID),
				},
				ServicePrincipalProfile: api.ServicePrincipalProfile{
					ClientID:     mockSubID,
					ClientSecret: "a",
				},
				NetworkProfile: api.NetworkProfile{
					PodCIDR:                "10.128.0.0/14",
					ServiceCIDR:            "172.30.0.0/16",
					SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
				},
				MasterProfile: api.MasterProfile{
					EncryptionAtHost: api.EncryptionAtHostDisabled,
					VMSize:           api.VMSizeStandardD8sV3,
					SubnetID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master", mockSubID),
				},
				FeatureProfile: api.FeatureProfile{
					GatewayEnabled: true,
				},
				WorkerProfiles: []api.WorkerProfile{
					{
						Name:             "worker",
						VMSize:           api.VMSizeStandardD16sV3,
						DiskSizeGB:       128,
						SubnetID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker", mockSubID),
						Count:            3,
						EncryptionAtHost: api.EncryptionAtHostDisabled,
					},
				},
				APIServerProfile: api.APIServerProfile{
					Visibility: api.VisibilityPublic,
				},
				IngressProfiles: []api.IngressProfile{
					{
						Name:       "default",
						Visibility: api.VisibilityPublic,
					},
				},
			},
		},
	}
}

func baseResponse() *v20200430.OpenShiftCluster {
	return &v20200430.OpenShiftCluster{
		ID:       testdatabase.GetResourcePath(mockSubID, "resourceName"),
		Name:     "resourceName",
		Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
		Location: "eastus",
		Properties: v20200430.OpenShiftClusterProperties{
			ClusterProfile: v20200430.ClusterProfile{
				Domain:          "example.eastus.aroapp.io",
				Version:         version.InstallStream.Version.String(),
				ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID),
			},
			ServicePrincipalProfile: v20200430.ServicePrincipalProfile{
				ClientID: mockSubID,
			},
			NetworkProfile: v20200430.NetworkProfile{
				PodCIDR:     "10.128.0.0/14",
				ServiceCIDR: "172.30.0.0/16",
			},
			MasterProfile: v20200430.MasterProfile{
				VMSize:   v20200430.VMSizeStandardD8sV3,
				SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/master", mockSubID),
			},
			WorkerProfiles: []v20200430.WorkerProfile{
				{
					Name:       "worker",
					VMSize:     v20200430.VMSizeStandardD16sV3,
					DiskSizeGB: 128,
					SubnetID:   fmt.Sprintf("/subscriptions/%s/resourceGroups/vnetResourceGroup/providers/Microsoft.Network/virtualNetworks/vnet/subnets/worker", mockSubID),
					Count:      3,
				},
			},
			APIServerProfile: v20200430.APIServerProfile{
				Visibility: v20200430.VisibilityPublic,
			},
			IngressProfiles: []v20200430.IngressProfile{
				{
					Name:       "default",
					Visibility: v20200430.VisibilityPublic,
				},
			},
		},
	}
}

func TestPutOrPatchOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"2020-04-30": api.APIs["2020-04-30"],
	}

	type test struct {
		name                   string
		request                func(*v20200430.OpenShiftCluster)
		isPatch                bool
		fixture                func(*testdatabase.Fixture)
		wantEnriched           []string
		wantSystemDataEnriched bool
		wantDocuments          func(*testdatabase.Checker)
		wantStatusCode         int
		wantResponse           func() *v20200430.OpenShiftCluster
		wantAsync              bool
		wantError              string
	}

	for _, tt := range []*test{
		{
			name: "create a new cluster",
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
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateCreating,
						ProvisioningState:        api.ProvisioningStateCreating,
					},
				})
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateCreating
				c.AddOpenShiftClusterDocuments(doc)
			},
			wantEnriched:   []string{},
			wantAsync:      true,
			wantStatusCode: http.StatusCreated,
			wantResponse: func() *v20200430.OpenShiftCluster {
				r := baseResponse()
				r.Properties.ProvisioningState = v20200430.ProvisioningStateCreating
				return r
			},
		},
		{
			name: "update a cluster from succeeded",
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
				doc := baseDocument()
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
				doc.OpenShiftCluster.Properties.LastProvisioningState = api.ProvisioningStateSucceeded
				c.AddOpenShiftClusterDocuments(doc)
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftCluster {
				r := baseResponse()
				r.Properties.ProvisioningState = v20200430.ProvisioningStateUpdating
				return r
			},
		},
		{
			name: "update a cluster from failed during update",
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
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateUpdating
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
				doc.OpenShiftCluster.Properties.LastProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateUpdating
				c.AddOpenShiftClusterDocuments(doc)
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftCluster {
				r := baseResponse()
				r.Properties.ProvisioningState = v20200430.ProvisioningStateUpdating
				return r
			},
		},
		{
			name: "update a cluster from failed during creation",
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
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateCreating
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name: "update a cluster from failed during deletion",
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
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateDeleting
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name:    "patch a cluster from succeeded",
			isPatch: true,
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
				doc := baseDocument()
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-be-kept"}
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})

				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
				doc.OpenShiftCluster.Properties.LastProvisioningState = api.ProvisioningStateSucceeded
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-be-kept"}
				c.AddOpenShiftClusterDocuments(doc)
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftCluster {
				r := baseResponse()
				r.Tags = map[string]string{"tag": "will-be-kept"}
				r.Properties.ProvisioningState = v20200430.ProvisioningStateUpdating
				return r
			},
		},
		{
			name:    "patch a cluster from failed during update",
			isPatch: true,
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
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateUpdating
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-be-kept"}
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantSystemDataEnriched: true,
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddAsyncOperationDocuments(&api.AsyncOperationDocument{
					OpenShiftClusterKey: strings.ToLower(testdatabase.GetResourcePath(mockSubID, "resourceName")),
					AsyncOperation: &api.AsyncOperation{
						InitialProvisioningState: api.ProvisioningStateUpdating,
						ProvisioningState:        api.ProvisioningStateUpdating,
					},
				})
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
				doc.OpenShiftCluster.Properties.LastProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateUpdating
				doc.OpenShiftCluster.Tags = map[string]string{"tag": "will-be-kept"}
				c.AddOpenShiftClusterDocuments(doc)
			},
			wantEnriched:   []string{testdatabase.GetResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftCluster {
				r := baseResponse()
				r.Tags = map[string]string{"tag": "will-be-kept"}
				r.Properties.ProvisioningState = v20200430.ProvisioningStateUpdating
				return r
			},
		},
		{
			name:    "patch a cluster from failed during creation",
			isPatch: true,
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
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateCreating
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name:    "patch a cluster from failed during deletion",
			isPatch: true,
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
				doc := baseDocument()
				doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateFailed
				doc.OpenShiftCluster.Properties.FailedProvisioningState = api.ProvisioningStateDeleting
				f.AddOpenShiftClusterDocuments(doc)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name: "creating cluster failing when provided cluster resource group already contains a cluster",
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
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(mockSubID, "otherResourceName")),
					ClusterResourceGroupIDKey: strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID)),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "otherResourceName"),
						Name:     "otherResourceName",
						Location: "eastus",
						Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Domain:               "example.eastus.aroapp.io",
								Version:              "4.3.0",
								ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID),
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateResourceGroup: : The provided resource group '/subscriptions/%s/resourcegroups/aro-vjb21wca' already contains a cluster.", mockSubID),
		},
		{
			name: "creating cluster failing when provided client ID is not unique",
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile.ClientID = mockSubID
			},
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
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:         strings.ToLower(testdatabase.GetResourcePath(mockSubID, "otherResourceName")),
					ClientIDKey: mockSubID,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:       testdatabase.GetResourcePath(mockSubID, "otherResourceName"),
						Name:     "otherResourceName",
						Location: "eastus",
						Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version:              "4.3.0",
								FipsValidatedModules: api.FipsValidatedModulesDisabled,
							},
							NetworkProfile: api.NetworkProfile{
								SoftwareDefinedNetwork: api.SoftwareDefinedNetworkOpenShiftSDN,
							},
							MasterProfile: api.MasterProfile{
								EncryptionAtHost: api.EncryptionAtHostDisabled,
							},
						},
					},
				})
			},
			wantSystemDataEnriched: true,
			wantAsync:              true,
			wantStatusCode:         http.StatusBadRequest,
			wantError:              fmt.Sprintf("400: DuplicateClientID: : The provided client ID '%s' is already in use by a cluster.", mockSubID),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).
				WithOpenShiftClusters().
				WithSubscriptions().
				WithAsyncOperations()
			defer ti.done()

			err := ti.buildFixtures(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, apis, &noop.Noop{}, nil, nil, nil, func(log *logrus.Entry, dialer proxy.Dialer, m metrics.Interface) clusterdata.OpenShiftClusterEnricher {
				return ti.enricher
			})
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).bucketAllocator = bucket.Fixed(1)
			f.(*frontend).now = func() time.Time { return mockCurrentTime }

			var systemDataEnricherCalled bool
			f.(*frontend).systemDataEnricher = func(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
				systemDataEnricherCalled = true
			}

			go f.Run(ctx, nil, nil)

			oc := &v20200430.OpenShiftCluster{
				Location: "eastus",
			}
			if tt.request != nil {
				tt.request(oc)
			}

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			} else {
				// if it's a put, fill in the rest of the properties
				oc.Properties = baseResponse().Properties
				oc.Properties.ServicePrincipalProfile.ClientSecret = "a"
			}

			resp, b, err := ti.request(method,
				"https://server"+testdatabase.GetResourcePath(mockSubID, "resourceName")+"?api-version=2020-04-30",
				http.Header{
					"Content-Type": []string{"application/json"},
				}, oc)
			if err != nil {
				t.Error(err)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("https://localhost:8443/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			var clusterResp *v20200430.OpenShiftCluster
			if tt.wantResponse != nil {
				clusterResp = tt.wantResponse()
			}

			errs := validateResponse(resp, b, tt.wantStatusCode, tt.wantError, clusterResp)
			for _, err := range errs {
				t.Error(err)
			}

			errs = ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}

			if tt.wantDocuments != nil {
				tt.wantDocuments(ti.checker)
				errs := ti.checker.CheckOpenShiftClusters(ti.openShiftClustersClient)
				errs = append(errs, ti.checker.CheckAsyncOperations(ti.asyncOperationsClient)...)
				for _, err := range errs {
					t.Error(err)
				}
			}

			if tt.wantSystemDataEnriched != systemDataEnricherCalled {
				t.Error(systemDataEnricherCalled)
			}
		})
	}
}

func TestEnrichSystemData(t *testing.T) {
	accountID1 := "00000000-0000-0000-0000-000000000001"
	accountID2 := "00000000-0000-0000-0000-000000000002"
	timestampString := "2021-01-23T12:34:54.0000000Z"
	timestamp, err := time.Parse(time.RFC3339, timestampString)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		systemData *api.SystemData
		expected   *api.OpenShiftClusterDocument
	}{
		{
			name:       "new systemData is nil",
			systemData: nil,
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			},
		},
		{
			name: "new systemData has all fields",
			systemData: &api.SystemData{
				CreatedBy:          accountID1,
				CreatedByType:      api.CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedBy:     accountID1,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						CreatedBy:          accountID1,
						CreatedByType:      api.CreatedByTypeApplication,
						CreatedAt:          &timestamp,
						LastModifiedBy:     accountID1,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
		{
			name: "update object",
			systemData: &api.SystemData{
				CreatedBy:          accountID1,
				CreatedByType:      api.CreatedByTypeApplication,
				CreatedAt:          &timestamp,
				LastModifiedBy:     accountID2,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						CreatedBy:          accountID1,
						CreatedByType:      api.CreatedByTypeApplication,
						CreatedAt:          &timestamp,
						LastModifiedBy:     accountID2,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
		{
			name: "old cluster update. Creation unknown",
			systemData: &api.SystemData{
				LastModifiedBy:     accountID2,
				LastModifiedByType: api.CreatedByTypeApplication,
				LastModifiedAt:     &timestamp,
			},
			expected: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					SystemData: api.SystemData{
						LastModifiedBy:     accountID2,
						LastModifiedByType: api.CreatedByTypeApplication,
						LastModifiedAt:     &timestamp,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			}
			enrichSystemData(doc, tt.systemData)

			if !reflect.DeepEqual(doc, tt.expected) {
				t.Error(cmp.Diff(doc, tt.expected))
			}
		})
	}
}
