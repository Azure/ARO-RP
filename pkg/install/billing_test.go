package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func TestCreateBillingEntry(t *testing.T) {
	ctx := context.Background()
	mockSubID := "11111111-1111-1111-1111-111111111111"
	mockTenantID := mockSubID
	location := "eastus"
	// controller := gomock.NewController(t)
	// defer controller.Finish()
	// billing := mock_database.NewMockBilling(controller)

	type test struct {
		name         string
		openshiftdoc *api.OpenShiftClusterDocument
		mocks        func(*test, *mock_database.MockBilling)
		wantError    error
	}

	for _, tt := range []*test{
		{
			name: "create a new billing entry",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       "11111111-1111-1111-1111-111111111111",
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
			mocks: func(tt *test, billing *mock_database.MockBilling) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID:        tt.openshiftdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
						Location:        tt.openshiftdoc.OpenShiftCluster.Location,
						CreationTime:    -1,
						LastBillingTime: -1,
					},
				}

				billing.EXPECT().
					Create(gomock.Any(), billingDoc).
					Return(billingDoc, nil)
			},
		},
		{
			name: "error on create a new billing entry",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       "11111111-1111-1111-1111-111111111111",
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
			mocks: func(tt *test, billing *mock_database.MockBilling) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID:        tt.openshiftdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
						Location:        tt.openshiftdoc.OpenShiftCluster.Location,
						CreationTime:    -1,
						LastBillingTime: -1,
					},
				}

				billing.EXPECT().
					Create(gomock.Any(), billingDoc).
					Return(nil, tt.wantError)
			},
			wantError: fmt.Errorf("Error creating document"),
		},
		{
			name: "update billing entry on create",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       "11111111-1111-1111-1111-111111111111",
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
			mocks: func(tt *test, billing *mock_database.MockBilling) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID:        tt.openshiftdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
						Location:        tt.openshiftdoc.OpenShiftCluster.Location,
						CreationTime:    -1,
						LastBillingTime: -1,
					},
				}

				billing.EXPECT().
					Create(gomock.Any(), billingDoc).
					Return(nil, &cosmosdb.Error{
						StatusCode: http.StatusConflict,
					})
				billing.EXPECT().
					Patch(gomock.Any(), billingDoc.ID, gomock.Any()).
					Return(nil, tt.wantError)
			},
			wantError: fmt.Errorf("Error creating document"),
		},
		{
			name: "error on update billing entry on create",
			openshiftdoc: &api.OpenShiftClusterDocument{
				Key:                       "11111111-1111-1111-1111-111111111111",
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
			mocks: func(tt *test, billing *mock_database.MockBilling) {
				billingDoc := &api.BillingDocument{
					Key:                       tt.openshiftdoc.Key,
					ClusterResourceGroupIDKey: tt.openshiftdoc.ClusterResourceGroupIDKey,
					ID:                        mockSubID,
					Billing: &api.Billing{
						TenantID:        tt.openshiftdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID,
						Location:        tt.openshiftdoc.OpenShiftCluster.Location,
						CreationTime:    -1,
						LastBillingTime: -1,
					},
				}

				billing.EXPECT().
					Create(gomock.Any(), billingDoc).
					Return(nil, &cosmosdb.Error{
						StatusCode: http.StatusConflict,
					})
				billing.EXPECT().
					Patch(gomock.Any(), billingDoc.ID, gomock.Any()).
					Return(billingDoc, nil)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			billing := mock_database.NewMockBilling(controller)

			tt.mocks(tt, billing)
			i := &Installer{
				log:     logrus.NewEntry(logrus.StandardLogger()),
				doc:     tt.openshiftdoc,
				billing: billing,
			}

			err := i.createBillingRecord(ctx)
			if err != nil {
				if tt.wantError != err {
					t.Errorf("Error want (%s), having (%s)", tt.wantError.Error(), err.Error())
				}
			}
		})
	}
}
