package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	hivev1alpha1 "github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
)

func Test_getAdminHiveClusterSync(t *testing.T) {
	fakeUUID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()
	clusterSync := hivev1alpha1.ClusterSync{Spec: hivev1alpha1.ClusterSyncSpec{}, Status: hivev1alpha1.ClusterSyncStatus{
		SyncSets: []hivev1alpha1.SyncStatus{{Name: "syncSet1", ObservedGeneration: 0, Result: "success", LastTransitionTime: metav1.Time{Time: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)}}},
	}}
	type test struct {
		name                            string
		resourceID                      string
		properties                      api.OpenShiftClusterProperties
		hiveEnabled                     bool
		expectedGetClusterSyncCallCount int
		wantStatusCode                  int
		wantResponse                    []byte
		wantError                       string
	}

	for _, tt := range []*test{
		{
			name:                            "cluster has hive profile with namespace",
			resourceID:                      fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/hive", fakeUUID),
			properties:                      api.OpenShiftClusterProperties{HiveProfile: api.HiveProfile{Namespace: fmt.Sprintf("aro-%s", fakeUUID)}},
			hiveEnabled:                     true,
			expectedGetClusterSyncCallCount: 1,
			wantResponse:                    []byte(`{"status":{"syncSets":[{"name":"syncSet1","observedGeneration":0,"result":"success","lastTransitionTime":"2024-07-01T00:00:00Z"}]}}`),
		},
		{
			name:                            "cluster does not have hive profile with namespace",
			resourceID:                      fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/nonHive", fakeUUID),
			hiveEnabled:                     true,
			expectedGetClusterSyncCallCount: 0,
			wantStatusCode:                  http.StatusNoContent,
			wantError:                       "204: ResourceNotFound: : cluster is not managed by hive",
		},
		{
			name:                            "hive is not enabled",
			resourceID:                      fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/nonHive", fakeUUID),
			hiveEnabled:                     false,
			expectedGetClusterSyncCallCount: 0,
			wantStatusCode:                  http.StatusBadRequest,
			wantError:                       "400: InvalidParameter: : hive is not enabled",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
			controller := gomock.NewController(t)
			defer ti.done()
			defer controller.Finish()

			ti.fixture.AddOpenShiftClusterDocuments(&api.OpenShiftClusterDocument{
				Key: strings.ToLower(tt.resourceID),
				OpenShiftCluster: &api.OpenShiftCluster{
					ID:         tt.resourceID,
					Name:       "hive",
					Type:       "Microsoft.RedHatOpenShift/openshiftClusters",
					Properties: tt.properties,
				},
			})

			err := ti.buildFixtures(nil)
			if err != nil {
				t.Fatal(err)
			}
			_env := ti.env.(*mock_env.MockInterface)
			var f *frontend
			if tt.hiveEnabled {
				clusterManager := mock_hive.NewMockClusterManager(controller)
				clusterManager.EXPECT().GetClusterSync(gomock.Any(), gomock.Any()).Return(&clusterSync, nil).Times(tt.expectedGetClusterSyncCallCount)
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, clusterManager, nil, nil, nil, nil)
			} else {
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, nil, nil, nil, nil, nil, nil)
			}

			if err != nil {
				t.Fatal(err)
			}
			hiveClusterSync, err := f._getAdminHiveClusterSync(ctx, strings.ToLower(tt.resourceID))
			cloudErr, isCloudErr := err.(*api.CloudError)
			if tt.wantError != "" && isCloudErr && cloudErr != nil {
				if tt.wantError != cloudErr.Error() {
					t.Errorf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
				if tt.wantStatusCode != 0 && tt.wantStatusCode != cloudErr.StatusCode {
					t.Errorf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
			}

			if !strings.EqualFold(string(hiveClusterSync), string(tt.wantResponse)) {
				t.Errorf("got %q and expected %q", hiveClusterSync, tt.wantResponse)
			}
		})
	}
}
