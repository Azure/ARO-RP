package billing

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

func TestIsSubscriptionRegisteredForE2E(t *testing.T) {
	mockSubID := "11111111-1111-1111-1111-111111111111"
	for _, tt := range []struct {
		name string
		sub  api.SubscriptionProperties
		want bool
	}{
		{
			name: "empty",
		},
		{
			name: "sub without feature flag registered and not internal tenant",
			sub: api.SubscriptionProperties{
				TenantID: mockSubID,
				RegisteredFeatures: []api.RegisteredFeatureProfile{
					{
						Name:  "RandomFeature",
						State: "Registered",
					},
				},
			},
		},
		{
			name: "sub with feature flag registered and not internal tenant",
			sub: api.SubscriptionProperties{
				TenantID: mockSubID,
				RegisteredFeatures: []api.RegisteredFeatureProfile{
					{
						Name:  featureSaveAROTestConfig,
						State: "Registered",
					},
				},
			},
		},
		{
			name: "AME internal tenant and feature flag not registered",
			sub: api.SubscriptionProperties{
				TenantID: tenantIDAME,
				RegisteredFeatures: []api.RegisteredFeatureProfile{
					{
						Name:  "RandomFeature",
						State: "Registered",
					},
				},
			},
		},
		{
			name: "MSFT internal tenant and feature flag not registered",
			sub: api.SubscriptionProperties{
				TenantID: tenantIDMSFT,
				RegisteredFeatures: []api.RegisteredFeatureProfile{
					{
						Name:  "RandomFeature",
						State: "Registered",
					},
				},
			},
		},
		{
			name: "AME internal tenant and feature flag registered",
			sub: api.SubscriptionProperties{
				TenantID: tenantIDAME,
				RegisteredFeatures: []api.RegisteredFeatureProfile{
					{
						Name:  featureSaveAROTestConfig,
						State: "Registered",
					},
				},
			},
			want: true,
		},
		{
			name: "MSFT internal tenant and feature flag registered",
			sub: api.SubscriptionProperties{
				TenantID: tenantIDMSFT,
				RegisteredFeatures: []api.RegisteredFeatureProfile{
					{
						Name:  featureSaveAROTestConfig,
						State: "Registered",
					},
				},
			},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubscriptionRegisteredForE2E(&tt.sub)
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	mockSubID := "11111111-1111-1111-1111-111111111111"
	mockTenantID := mockSubID
	location := "eastus"

	type test struct {
		name         string
		openshiftdoc *api.OpenShiftClusterDocument
		mocks        func(*test, *mock_database.MockBilling, *mock_database.MockSubscriptions)
		wantErr      string
	}

	// Can't add tests for billing storage because there isn't an interface on
	// the azure storage clients.

	for _, tt := range []*test{
		{
			name: "successful mark for deletion on billing entity, with a subscription not registered for e2e",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName/providers/microsoft.redhatopenshift/openshiftclusters/clusterName", mockSubID),
				ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName", mockSubID),
				ID:                        mockSubID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: api.ServicePrincipalProfile{
							TenantID: mockTenantID,
						},
					},
					Location: location,
				},
			},
			mocks: func(tt *test, billing *mock_database.MockBilling, subscription *mock_database.MockSubscriptions) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID: mockTenantID,
						Location: location,
					},
				}

				billing.EXPECT().
					MarkForDeletion(gomock.Any(), billingDoc.ID).
					Return(billingDoc, nil)

				subscription.EXPECT().
					Get(gomock.Any(), mockSubID).
					Return(&api.SubscriptionDocument{
						Subscription: &api.Subscription{
							Properties: &api.SubscriptionProperties{
								RegisteredFeatures: []api.RegisteredFeatureProfile{
									{
										Name:  featureSaveAROTestConfig,
										State: "NotRegistered",
									},
								},
							},
						},
					}, nil)
			},
		},
		{
			name: "no error on mark for deletion on billing entry that is not found",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName/providers/microsoft.redhatopenshift/openshiftclusters/clusterName", mockSubID),
				ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName", mockSubID),
				ID:                        mockSubID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: api.ServicePrincipalProfile{
							TenantID: mockTenantID,
						},
					},
					Location: location,
				},
			},
			mocks: func(tt *test, billing *mock_database.MockBilling, subscription *mock_database.MockSubscriptions) {
				billing.EXPECT().
					MarkForDeletion(gomock.Any(), tt.openshiftdoc.ID).
					Return(nil, &cosmosdb.Error{
						StatusCode: 404,
					})
			},
		},
		{
			name: "error on mark for deletion on billing entry",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName/providers/microsoft.redhatopenshift/openshiftclusters/clusterName", mockSubID),
				ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName", mockSubID),
				ID:                        mockSubID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: api.ServicePrincipalProfile{
							TenantID: mockTenantID,
						},
					},
					Location: location,
				},
			},
			mocks: func(tt *test, billing *mock_database.MockBilling, subscription *mock_database.MockSubscriptions) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID: tt.openshiftdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
						Location: tt.openshiftdoc.OpenShiftCluster.Location,
					},
				}

				billing.EXPECT().
					MarkForDeletion(gomock.Any(), tt.openshiftdoc.ID).
					Return(billingDoc, errors.New("random error"))
			},
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().DeploymentMode().AnyTimes().Return(deployment.Production)

			log := logrus.NewEntry(logrus.StandardLogger())

			billingDB := mock_database.NewMockBilling(controller)
			subsDB := mock_database.NewMockSubscriptions(controller)

			tt.mocks(tt, billingDB, subsDB)

			m := &manager{
				log:       log,
				billingDB: billingDB,
				subDB:     subsDB,
				env:       _env,
			}

			err := m.Delete(ctx, tt.openshiftdoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestEnsure(t *testing.T) {
	ctx := context.Background()
	mockSubID := "11111111-1111-1111-1111-111111111111"
	mockTenantID := mockSubID
	mockInfraID := "infra"
	location := "eastus"

	type test struct {
		name         string
		openshiftdoc *api.OpenShiftClusterDocument
		mocks        func(*test, *mock_database.MockBilling, *mock_database.MockSubscriptions)
		wantErr      string
	}

	// Can't add tests for billing storage because there isn't an interface on
	// the azure storage clients.

	for _, tt := range []*test{
		{
			name: "create a new billing entry with a subscription not registered for e2e",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName/providers/microsoft.redhatopenshift/openshiftclusters/clusterName", mockSubID),
				ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName", mockSubID),
				ID:                        mockSubID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: api.ServicePrincipalProfile{
							TenantID: mockTenantID,
						},
						InfraID: mockInfraID,
					},
					Location: location,
				},
			},
			mocks: func(tt *test, billing *mock_database.MockBilling, subscription *mock_database.MockSubscriptions) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID: mockTenantID,
						Location: location,
					},
					InfraID: mockInfraID,
				}

				billing.EXPECT().
					Create(gomock.Any(), billingDoc).
					Return(billingDoc, nil)

				subscription.EXPECT().
					Get(gomock.Any(), mockSubID).
					Return(&api.SubscriptionDocument{
						Subscription: &api.Subscription{
							Properties: &api.SubscriptionProperties{
								RegisteredFeatures: []api.RegisteredFeatureProfile{
									{
										Name:  featureSaveAROTestConfig,
										State: "NotRegistered",
									},
								},
							},
						},
					}, nil)
			},
		},
		{
			name: "error on create a new billing entry",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName/providers/microsoft.redhatopenshift/openshiftclusters/clusterName", mockSubID),
				ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName", mockSubID),
				ID:                        mockSubID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: api.ServicePrincipalProfile{
							TenantID: mockTenantID,
						},
						InfraID: mockInfraID,
					},
					Location: location,
				},
			},
			mocks: func(tt *test, billing *mock_database.MockBilling, subscription *mock_database.MockSubscriptions) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID: tt.openshiftdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
						Location: tt.openshiftdoc.OpenShiftCluster.Location,
					},
					InfraID: mockInfraID,
				}

				billing.EXPECT().
					Create(gomock.Any(), billingDoc).
					Return(nil, errors.New("random error"))
			},
			wantErr: "random error",
		},
		{
			name: "billing document already existing on DB on create",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName/providers/microsoft.redhatopenshift/openshiftclusters/clusterName", mockSubID),
				ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName", mockSubID),
				ID:                        mockSubID,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: api.ServicePrincipalProfile{
							TenantID: mockTenantID,
						},
						InfraID: mockInfraID,
					},
					Location: location,
				},
			},
			mocks: func(tt *test, billing *mock_database.MockBilling, subscription *mock_database.MockSubscriptions) {
				billingDoc := &api.BillingDocument{
					Key:                       fmt.Sprintf("/subscriptions/%s/resourcegroups/rgName/providers/microsoft.redhatopenshift/openshiftclusters/clusterName", mockSubID),
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID: tt.openshiftdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
						Location: tt.openshiftdoc.OpenShiftCluster.Location,
					},
					InfraID: mockInfraID,
				}

				billing.EXPECT().
					Create(gomock.Any(), billingDoc).
					Return(nil, &cosmosdb.Error{
						StatusCode: http.StatusConflict,
					})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().DeploymentMode().AnyTimes().Return(deployment.Production)

			log := logrus.NewEntry(logrus.StandardLogger())

			billingDB := mock_database.NewMockBilling(controller)
			subsDB := mock_database.NewMockSubscriptions(controller)

			tt.mocks(tt, billingDB, subsDB)

			m := &manager{
				log:       log,
				billingDB: billingDB,
				subDB:     subsDB,
				env:       _env,
			}

			err := m.Ensure(ctx, tt.openshiftdoc)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}
