package billing

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestDelete(t *testing.T) {
	ctx := context.Background()

	const (
		docID    = "00000000-0000-0000-0000-000000000000"
		subID    = "11111111-1111-1111-1111-111111111111"
		tenantID = "22222222-2222-2222-2222-222222222222"
		location = "eastus"
	)

	type test struct {
		name          string
		fixture       func(*testdatabase.Fixture)
		wantDocuments func(*testdatabase.Checker)
		dbError       error
		wantErr       string
	}

	// Can't add tests for billing storage because there isn't an interface on
	// the azure storage clients.

	for _, tt := range []*test{
		{
			name: "successful mark for deletion on billing entity, with a subscription not registered for e2e",
			fixture: func(f *testdatabase.Fixture) {
				f.AddBillingDocuments(&api.BillingDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					Billing: &api.Billing{
						TenantID: tenantID,
						Location: location,
					},
				})
			},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddBillingDocuments(&api.BillingDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					Billing: &api.Billing{
						TenantID:     tenantID,
						Location:     location,
						DeletionTime: 1,
					},
				})
			},
		},
		{
			name: "no error on mark for deletion on billing entry that is not found",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
					},
				})
			},
		},
		{
			name: "error on mark for deletion on billing entry",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Location: location,
					},
				})
			},
			dbError: errors.New("random error"),
			wantErr: "random error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			controller := gomock.NewController(t)
			defer controller.Finish()

			log := logrus.NewEntry(logrus.StandardLogger())
			openShiftClusterDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			billingDatabase, billingClient := testdatabase.NewFakeBilling()
			subscriptionsDatabase, _ := testdatabase.NewFakeSubscriptions()

			if tt.fixture != nil {
				fixture := testdatabase.NewFixture().
					WithOpenShiftClusters(openShiftClusterDatabase).
					WithBilling(billingDatabase).
					WithSubscriptions(subscriptionsDatabase)
				tt.fixture(fixture)
				err = fixture.Create()
				if err != nil {
					t.Fatal(err)
				}
			}

			if tt.dbError != nil {
				billingClient.SetError(tt.dbError)
			}

			m := &manager{
				log:       log,
				billingDB: billingDatabase,
			}

			err = m.Delete(ctx, &api.OpenShiftClusterDocument{ID: docID})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantDocuments != nil {
				checker := testdatabase.NewChecker()
				tt.wantDocuments(checker)
				errs := checker.CheckBilling(billingClient)
				for _, err := range errs {
					t.Error(err)
				}
			}
		})
	}
}

func TestEnsure(t *testing.T) {
	ctx := context.Background()

	const (
		docID       = "00000000-0000-0000-0000-000000000000"
		subID       = "11111111-1111-1111-1111-111111111111"
		tenantID    = "22222222-2222-2222-2222-222222222222"
		mockInfraID = "infra"
		location    = "eastus"
	)

	type test struct {
		name          string
		fixture       func(*testdatabase.Fixture)
		wantDocuments func(*testdatabase.Checker)
		dbError       error
		wantErr       string
	}

	// Can't add tests for billing storage because there isn't an interface on
	// the azure storage clients.

	for _, tt := range []*test{
		{
			name: "create a new billing entry with a subscription not registered for e2e",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							InfraID: mockInfraID,
						},
						Location: location,
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: subID,
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							TenantID: tenantID,
						},
					},
				})
			},
			wantDocuments: func(c *testdatabase.Checker) {
				c.AddBillingDocuments(&api.BillingDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					Billing: &api.Billing{
						TenantID: tenantID,
						Location: location,
					},
					InfraID: mockInfraID,
				})
			},
		},
		{
			name: "error on create a new billing entry",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							InfraID: mockInfraID,
						},
						Location: location,
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: subID,
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							TenantID: tenantID,
						},
					},
				})
			},
			dbError: errors.New("random error"),
			wantErr: "random error",
		},
		{
			name: "billing document already existing on DB on create",
			fixture: func(f *testdatabase.Fixture) {
				f.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
					Key:                       strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							InfraID: mockInfraID,
						},
						Location: location,
					},
				})
				f.AddSubscriptionDocuments(&api.SubscriptionDocument{
					ID: subID,
					Subscription: &api.Subscription{
						Properties: &api.SubscriptionProperties{
							TenantID: tenantID,
						},
					},
				})
				f.AddBillingDocuments(&api.BillingDocument{
					Key:                       testdatabase.GetResourcePath(subID, "resourceName"),
					ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup", subID),
					ID:                        docID,
					Billing: &api.Billing{
						TenantID: tenantID,
						Location: location,
					},
					InfraID: mockInfraID,
				})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			controller := gomock.NewController(t)
			defer controller.Finish()

			_env := mock_env.NewMockInterface(controller)
			_env.EXPECT().IsLocalDevelopmentMode().AnyTimes().Return(false)

			log := logrus.NewEntry(logrus.StandardLogger())
			openShiftClusterDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			billingDatabase, billingClient := testdatabase.NewFakeBilling()
			subscriptionsDatabase, _ := testdatabase.NewFakeSubscriptions()

			if tt.fixture != nil {
				fixture := testdatabase.NewFixture().
					WithOpenShiftClusters(openShiftClusterDatabase).
					WithBilling(billingDatabase).
					WithSubscriptions(subscriptionsDatabase)
				tt.fixture(fixture)
				err = fixture.Create()
				if err != nil {
					t.Fatal(err)
				}
			}

			if tt.dbError != nil {
				billingClient.SetError(tt.dbError)
			}

			m := &manager{
				log:       log,
				billingDB: billingDatabase,
			}

			doc, err := openShiftClusterDatabase.Get(ctx, strings.ToLower(testdatabase.GetResourcePath(subID, "resourceName")))
			if err != nil {
				t.Fatal(err)
			}
			subDoc, err := subscriptionsDatabase.Get(ctx, subID)
			if err != nil {
				t.Fatal(err)
			}
			err = m.Ensure(ctx, doc, subDoc)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantDocuments != nil {
				checker := testdatabase.NewChecker()
				tt.wantDocuments(checker)
				errs := checker.CheckBilling(billingClient)
				for _, err := range errs {
					t.Error(err)
				}
			}
		})
	}
}
