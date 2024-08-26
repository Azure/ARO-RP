package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
)

func TestGetAdminHiveSyncsetResources(t *testing.T) {
	fakeNamespace := "aro-00000000-0000-0000-0000-000000000000"
	ctx := context.Background()
	clusterSyncsetTest := &v1alpha1.ClusterSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clustersync1",
			Namespace: fakeNamespace,
		},
	}

	type test struct {
		name           string
		namespace      string
		hiveEnabled    bool
		mocks          func(*test, *mock_hive.MockClusterManager)
		wantStatusCode int
		wantResponse   []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:           "Cluster SyncSets must be namespaced",
			namespace:      "",
			hiveEnabled:    true,
			mocks:          func(tt *test, s *mock_hive.MockClusterManager) {},
			wantStatusCode: http.StatusNotFound,
			wantError:      "404: NotFound: : cluster not found",
		},
		{
			name:      "List ClusterSync resources successfully",
			namespace: "hive",
			wantError: "",
			mocks: func(tt *test, s *mock_hive.MockClusterManager) {
				s.EXPECT().
					GetSyncSetResources(gomock.Any(), gomock.Any()).
					Return(&clusterSyncsetTest, nil).Times(1)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Hive is not enabled",
			namespace:      fakeNamespace,
			mocks:          nil,
			hiveEnabled:    false,
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "500: InternalServerError: : hive is not enabled",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			defer ti.done()

			_env := ti.env.(*mock_env.MockInterface)
			var f *frontend
			var err error
			if tt.hiveEnabled {
				s := mock_hive.NewMockClusterManager(ti.controller) //NewMockSyncSetResourceManager(ti.controller)
				tt.mocks(tt, s)
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil)
			} else {
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil)
			}
			if err != nil {
				t.Fatal(err)
			}

			clusterSyncSet, err := f._getAdminHiveSyncsetResources(ctx, tt.namespace)
			cloudErr, isCloudErr := err.(*api.CloudError)
			if tt.wantError != "" && isCloudErr && cloudErr != nil {
				if tt.wantError != cloudErr.Error() {
					t.Fatalf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
				if tt.wantStatusCode != 0 && tt.wantStatusCode != cloudErr.StatusCode {
					t.Fatalf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
			}

			if !strings.EqualFold(string(clusterSyncSet), string(tt.wantResponse)) {
				t.Fatalf("got %q and expected %q", clusterSyncSet, tt.wantResponse)
			}
		})
	}
}
