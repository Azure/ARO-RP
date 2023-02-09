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
	hivev1 "github.com/openshift/hive/apis/hive/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
)

func Test_getAdminHiveClusterDeployment(t *testing.T) {
	fakeUUID := "00000000-0000-0000-0000-000000000000"
	ctx := context.Background()
	clusterDeployment := hivev1.ClusterDeployment{
		Spec: hivev1.ClusterDeploymentSpec{
			ClusterName: "abc123",
		},
	}
	type test struct {
		name                                  string
		resourceID                            string
		properties                            api.OpenShiftClusterProperties
		hiveEnabled                           bool
		expectedGetClusterDeploymentCallCount int
		wantStatusCode                        int
		wantResponse                          []byte
		wantError                             string
	}

	for _, tt := range []*test{
		{
			name:                                  "cluster has hive profile with namespace",
			resourceID:                            fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/hive", fakeUUID),
			properties:                            api.OpenShiftClusterProperties{HiveProfile: api.HiveProfile{Namespace: fmt.Sprintf("aro-%s", fakeUUID)}},
			hiveEnabled:                           true,
			expectedGetClusterDeploymentCallCount: 1,
			wantResponse:                          []byte(`{"spec":{"clusterName":"abc123","baseDomain":"","platform":{},"installed":false}}`),
		},
		{
			name:                                  "cluster does not have hive profile with namespace",
			resourceID:                            fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/nonHive", fakeUUID),
			hiveEnabled:                           true,
			expectedGetClusterDeploymentCallCount: 0,
			wantStatusCode:                        http.StatusNoContent,
			wantError:                             "204: ResourceNotFound: : cluster is not managed by hive",
		},
		{
			name:                                  "hive is not enabled",
			resourceID:                            fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/nonHive", fakeUUID),
			hiveEnabled:                           false,
			expectedGetClusterDeploymentCallCount: 0,
			wantStatusCode:                        http.StatusInternalServerError,
			wantError:                             "500: InternalServerError: : hive is not enabled",
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
				clusterManager.EXPECT().GetClusterDeployment(gomock.Any(), gomock.Any()).Return(&clusterDeployment, nil).Times(tt.expectedGetClusterDeploymentCallCount)
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase,
					ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, clusterManager, nil, nil, nil)
			} else {
				f, err = NewFrontend(ctx, ti.audit, ti.log, _env, ti.asyncOperationsDatabase, ti.clusterManagerDatabase, ti.openShiftClustersDatabase,
					ti.subscriptionsDatabase, nil, api.APIs, &noop.Noop{}, nil, nil, nil, nil, nil)
			}

			if err != nil {
				t.Fatal(err)
			}
			hiveClusterDeployment, err := f._getAdminHiveClusterDeployment(ctx, strings.ToLower(tt.resourceID))
			cloudErr, isCloudErr := err.(*api.CloudError)
			if tt.wantError != "" && isCloudErr && cloudErr != nil {
				if tt.wantError != cloudErr.Error() {
					t.Fatalf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
				if tt.wantStatusCode != 0 && tt.wantStatusCode != cloudErr.StatusCode {
					t.Fatalf("got %q but wanted %q", cloudErr.Error(), tt.wantError)
				}
			}

			if !strings.EqualFold(string(hiveClusterDeployment), string(tt.wantResponse)) {
				t.Fatalf("got %q and expected %q", hiveClusterDeployment, tt.wantResponse)
			}
		})
	}
}
