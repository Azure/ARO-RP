package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_privatedns "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/mgmt/privatedns"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestRemovePrivateDNSZone(t *testing.T) {
	ctx := context.Background()
	const resourceGroupID = "/subscriptions/0000000-0000-0000-0000-000000000000/resourceGroups/testGroup"

	t.Run("should return nil when privateZones.ListByResourceGroup() returns an error", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()

		privateZones := mock_privatedns.NewMockPrivateZonesClient(controller)
		privateZones.EXPECT().ListByResourceGroup(ctx, gomock.Any(), nil).Return(nil, errors.New("someError"))

		doc := &api.OpenShiftClusterDocument{
			OpenShiftCluster: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					ClusterProfile: api.ClusterProfile{
						ResourceGroupID: resourceGroupID,
					},
				},
			},
		}
		m := &manager{
			doc:          doc,
			log:          logrus.NewEntry(logrus.StandardLogger()),
			privateZones: privateZones,
		}

		err := m.removePrivateDNSZone(ctx)

		expectedError := ""
		utilerror.AssertErrorMessage(t, err, expectedError)
	})
}
