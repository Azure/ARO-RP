package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
)

func TestAdminRedeployVM(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"

	ctx := context.Background()

	type test struct {
		name           string
		resourceID     string
		vmName         string
		mocks          func(*test, *mock_adminactions.MockInterface, *mock_database.MockOpenShiftClusters, *mock_database.MockSubscriptions)
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			vmName:     "aro-worker-australiasoutheast-7tcq7",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, a *mock_adminactions.MockInterface, oc *mock_database.MockOpenShiftClusters, s *mock_database.MockSubscriptions) {
				clusterDoc := &api.OpenShiftClusterDocument{
					Key: tt.resourceID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				}
				subscriptionDoc := &api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				}

				oc.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(clusterDoc, nil)

				s.EXPECT().Get(gomock.Any(), strings.ToLower(mockSubID)).
					Return(subscriptionDoc, nil)

				a.EXPECT().VMRedeployAndWait(gomock.Any(), tt.vmName).Return(nil)

			},
			wantStatusCode: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti, err := newTestInfra(t)
			if err != nil {
				t.Fatal(err)
			}
			defer ti.done()

			a := mock_adminactions.NewMockInterface(ti.controller)
			dbopenshiftclusters := mock_database.NewMockOpenShiftClusters(ti.controller)
			dbsubscriptions := mock_database.NewMockSubscriptions(ti.controller)
			tt.mocks(tt, a, dbopenshiftclusters, dbsubscriptions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, nil, nil, nil, dbopenshiftclusters, dbsubscriptions, ti.l, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, env.FPAuthorizer, proxy.Dialer, *api.OpenShiftCluster,
				*api.SubscriptionDocument) (adminactions.Interface, error) {
				return a, nil
			}, nil, clientauthorizer.NewOne(clientcerts[0].Raw))

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server/admin%s/redeployvm?vmName=%s", tt.resourceID, tt.vmName),
				nil, nil)

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
