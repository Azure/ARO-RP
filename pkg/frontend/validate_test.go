package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/test/util/matcher"
)

func TestValidate(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name       string
		resourceID string
		mocks      func(*test, *mock_database.MockOpenShiftClusters)
		dbGetDoc   *api.OpenShiftClusterDocument
		dbGetErr   error
		wantError  *api.CloudError
	}

	for _, tt := range []*test{
		{
			name:       "validateOpenShiftUniqueKey",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openshiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
				clusterdoc := &api.OpenShiftClusterDocument{
					Key:    strings.ToLower(tt.resourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.Properties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version: "4.3.0",
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								TenantID: "11111111-1111-1111-1111-111111111111",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Create(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantError: api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error."),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			controller := gomock.NewController(t)
			defer controller.Finish()

			openShiftClusters := mock_database.NewMockOpenShiftClusters(controller)

			tt.mocks(tt, openShiftClusters)

			if tt.wantError == nil {
				// No error
			} else {
				// Error
			}
		})
	}

}
