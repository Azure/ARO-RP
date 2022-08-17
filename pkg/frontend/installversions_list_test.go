package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20220904 "github.com/Azure/ARO-RP/pkg/api/v20220904"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestListInstallVersions(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"
	method := http.MethodGet
	ctx := context.Background()

	type test struct {
		name                    string
		mocks                   func(*test, *mock_adminactions.MockKubeActions)
		wantStatusCode          int
		wantResponse            *v20220904.InstallVersions
		wantResponseContentType string
		wantError               string
	}

	for _, tt := range []*test{
		{
			name:                    "hard coded versions",
			mocks:                   func(t *test, mka *mock_adminactions.MockKubeActions) {},
			wantStatusCode:          http.StatusOK,
			wantResponseContentType: "application/json",
			wantResponse:            &v20220904.InstallVersions{version.InstallStream.Version.String()},
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", method, tt.name), func(t *testing.T) {
			ti := newTestInfra(t).WithSubscriptions()
			defer ti.done()
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
				return nil, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(method,
				fmt.Sprintf("https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/listinstallversions?api-version=2022-09-04", mockSubID, ti.env.Location()),
				nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			b = bytes.TrimSpace(b)

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
			if tt.wantResponseContentType != resp.Header.Get("Content-Type") {
				t.Error(fmt.Errorf("unexpected \"Content-Type\" response header value \"%s\", wanted \"%s\"", resp.Header.Get("Content-Type"), tt.wantResponseContentType))
			}
		})
	}
}
