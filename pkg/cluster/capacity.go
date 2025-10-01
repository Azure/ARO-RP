package cluster

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) ensureControlPlaneCapacity(ctx context.Context) error {
	location := m.doc.OpenShiftCluster.Location
	resourceGroupName := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	capacityGroupName := "aro-crg"

	m.log.Info("creating capacity reservation group")
	_, err := m.armCapacityReservationGroups.CreateOrUpdate(
		ctx, resourceGroupName, capacityGroupName, armcompute.CapacityReservationGroup{
			Location: pointerutils.ToPtr(location),
			Zones:    pointerutils.ToSlicePtr(m.doc.OpenShiftCluster.Properties.Zones),
		}, nil)
	if err != nil {
		return err
	}

	reservationName := "controlplane-" + m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize.String()

	m.log.Info("creating capacity reservation")
	err = m.armCapacityReservations.CreateOrUpdateAndWait(
		ctx, resourceGroupName, capacityGroupName, reservationName, armcompute.CapacityReservation{
			Location: pointerutils.ToPtr(location),
			SKU: &armcompute.SKU{
				Name:     pointerutils.ToPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize.String()),
				Capacity: pointerutils.ToPtr(int64(3)),
			},
			Zones: pointerutils.ToSlicePtr(m.doc.OpenShiftCluster.Properties.Zones),
		}, nil)

	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			return &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeDeploymentFailed,
					Message: "Insufficient capacity in " + m.doc.OpenShiftCluster.Properties.MasterProfile.VMSize.String(),
					Details: []api.CloudErrorBody{
						{
							Code:    api.CloudErrorCodeDeploymentFailed,
							Message: err.Error(),
						},
					},
				},
			}
		} else {
			return err
		}
	}

	// Add the capacity reservation info to the document
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ClusterProfile.CapacityReservationGroup = capacityGroupName
		doc.OpenShiftCluster.Properties.MasterProfile.CapacityReservationName = reservationName
		return nil
	})
	return err
}
