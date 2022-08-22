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
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
)

func TestAdminApproveCSR(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()

	type test struct {
		name       string
		resourceID string
		csrName    string
		// approveAll     string
		mocks          func(*test, *mock_adminactions.MockKubeActions)
		method         string
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			method:     http.MethodPost,
			name:       "single csr",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			csrName:    "aro-csr",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					RunCertificateApprove(gomock.Any(), tt.csrName).
					Return(nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			method:     http.MethodPost,
			name:       "all csrs",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			csrName:    "aro-csr",
			approveAll: "true",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
				k.EXPECT().
					RunCertificateMassApprove(gomock.Any()).
					Return(nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			method:     http.MethodPost,
			name:       "invalid csr name",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			csrName:    "",
			approveAll: "false",
			mocks: func(tt *test, k *mock_adminactions.MockKubeActions) {
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided name '' is invalid.",
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", tt.method, tt.name), func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			k := mock_adminactions.NewMockKubeActions(ti.controller)
			tt.mocks(tt, k)

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(tt.resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
				},
			})
			ti.fixture.AddSubscriptionDocuments(&api.SubscriptionDocument{
				ID: mockSubID,
				Subscription: &api.Subscription{
					State: api.SubscriptionStateRegistered,
					Properties: &api.SubscriptionProperties{
						TenantID: mockTenantID,
					},
				},
			})

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewFrontend(ctx, ti.audit, ti.log, ti.env, ti.asyncOperationsDatabase, ti.openShiftClustersDatabase, ti.subscriptionsDatabase, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
				return k, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(tt.method,
				fmt.Sprintf("https://server/admin%s/approvecsr?csrName=%s&approveAll=%s", tt.resourceID, tt.csrName, tt.approveAll),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
