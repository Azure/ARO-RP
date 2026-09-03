package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const testCRGName = "aro-resize-crg-cp-test"

func TestCRGSetupForResize(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()

	clusterRG := "cluster-rg"
	location := "eastus"
	targetSKU := "Standard_D16s_v3"
	threeZones := []string{"1", "2", "3"}
	crgID := "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/" + testCRGName

	allocationFailed := &azcore.ResponseError{ErrorCode: "AllocationFailed", StatusCode: http.StatusConflict}
	conflict409 := &azcore.ResponseError{ErrorCode: "Conflict", StatusCode: http.StatusConflict}
	invalidName := &azcore.ResponseError{ErrorCode: "InvalidResourceName", StatusCode: http.StatusBadRequest}

	for _, tt := range []struct {
		name        string
		zones       []string
		topology    zoneTopology
		mocks       func(*mock_adminactions.MockAzureActions)
		wantCreated bool
		wantErr     string
	}{
		{
			name:     "three-zone happy path",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, threeZones, testCRGName).Return(crgID, nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, threeZones[0], targetSKU, testCRGName, int64(1)).Return(nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, threeZones[1], targetSKU, testCRGName, int64(1)).Return(nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, threeZones[2], targetSKU, testCRGName, int64(1)).Return(nil)
			},
			wantCreated: true,
		},
		{
			name:     "regional happy path",
			zones:    nil,
			topology: zoneTopologyRegional,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, gomock.Nil(), testCRGName).Return(crgID, nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, "", targetSKU, testCRGName, int64(3)).Return(nil)
			},
			wantCreated: true,
		},
		{
			name:     "CRG create fails",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, threeZones, testCRGName).Return("", errors.New("network error"))
			},
			wantCreated: false,
			wantErr:     "creating capacity reservation group: network error",
		},
		{
			name:     "CRG name rejected by Azure - 400",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, threeZones, testCRGName).Return("", invalidName)
			},
			wantCreated: false,
			wantErr:     "creating capacity reservation group: " + invalidName.Error(),
		},
		{
			name:     "CRG already exists - conflict from prior failed resize",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, threeZones, testCRGName).Return("", conflict409)
			},
			wantCreated: false,
			wantErr:     "creating capacity reservation group: " + conflict409.Error(),
		},
		{
			name:     "three-zone reservation fails with capacity error - returns true so caller can teardown",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, threeZones, testCRGName).Return(crgID, nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, threeZones[0], targetSKU, testCRGName, int64(1)).Return(allocationFailed)
			},
			wantCreated: true,
			wantErr:     "creating capacity reservation in zone " + threeZones[0] + ": " + allocationFailed.Error(),
		},
		{
			name:     "zone capacity held by concurrent resize - returns true so caller can teardown",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, threeZones, testCRGName).Return(crgID, nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, threeZones[0], targetSKU, testCRGName, int64(1)).Return(conflict409)
			},
			wantCreated: true,
			wantErr:     "creating capacity reservation in zone " + threeZones[0] + ": " + conflict409.Error(),
		},
		{
			name:     "regional reservation fails - returns true so caller can teardown",
			zones:    nil,
			topology: zoneTopologyRegional,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, gomock.Nil(), testCRGName).Return(crgID, nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, "", targetSKU, testCRGName, int64(3)).Return(allocationFailed)
			},
			wantCreated: true,
			wantErr:     "creating regional capacity reservation: " + allocationFailed.Error(),
		},
		{
			name:     "stops at first reservation failure - remaining zones not called",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().CreateCRG(gomock.Any(), clusterRG, location, threeZones, testCRGName).Return(crgID, nil)
				a.EXPECT().CreateCapacityReservation(gomock.Any(), clusterRG, location, threeZones[0], targetSKU, testCRGName, int64(1)).
					Return(errors.New("zone 1 failed"))
				// zones[1] and zones[2] must NOT be called — gomock will fail the test if they are
			},
			wantCreated: true,
			wantErr:     "creating capacity reservation in zone " + threeZones[0] + ": zone 1 failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			a := mock_adminactions.NewMockAzureActions(ctrl)
			tt.mocks(a)

			crgCreated, err := crgSetupForResize(ctx, log, a, clusterRG, location, tt.zones, tt.topology, targetSKU, testCRGName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if crgCreated != tt.wantCreated {
				t.Errorf("crgCreated = %v, want %v", crgCreated, tt.wantCreated)
			}
		})
	}
}

func TestCRGTeardown(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()

	clusterRG := "cluster-rg"
	threeZones := []string{"1", "2", "3"}

	for _, tt := range []struct {
		name     string
		zones    []string
		topology zoneTopology
		mocks    func(*mock_adminactions.MockAzureActions)
		wantErr  string
	}{
		{
			name:     "three-zone happy path",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[0]).Return(nil)
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[1]).Return(nil)
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[2]).Return(nil)
				a.EXPECT().DeleteCRG(gomock.Any(), clusterRG, testCRGName).Return(nil)
			},
		},
		{
			name:     "regional happy path",
			zones:    nil,
			topology: zoneTopologyRegional,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, "").Return(nil)
				a.EXPECT().DeleteCRG(gomock.Any(), clusterRG, testCRGName).Return(nil)
			},
		},
		{
			name:     "reservation delete fails - continues remaining zones then skips CRG delete",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[0]).Return(errors.New("zone 1 failed"))
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[1]).Return(nil)
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[2]).Return(nil)
			},
			wantErr: "zone 1 failed",
		},
		{
			name:     "multiple reservation deletes fail - all errors joined",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[0]).Return(errors.New("zone 1 failed"))
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[1]).Return(errors.New("zone 2 failed"))
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[2]).Return(nil)
			},
			wantErr: "zone 1 failed\nzone 2 failed",
		},
		{
			name:     "regional reservation delete fails - skips CRG delete",
			zones:    nil,
			topology: zoneTopologyRegional,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, "").Return(errors.New("regional delete failed"))
			},
			wantErr: "regional delete failed",
		},
		{
			name:     "CRG delete fails after all reservations removed",
			zones:    threeZones,
			topology: zoneTopologyThreeZone,
			mocks: func(a *mock_adminactions.MockAzureActions) {
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[0]).Return(nil)
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[1]).Return(nil)
				a.EXPECT().DeleteCapacityReservation(gomock.Any(), clusterRG, testCRGName, threeZones[2]).Return(nil)
				a.EXPECT().DeleteCRG(gomock.Any(), clusterRG, testCRGName).Return(errors.New("CRG delete failed"))
			},
			wantErr: "CRG delete failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			a := mock_adminactions.NewMockAzureActions(ctrl)
			tt.mocks(a)

			err := crgTeardown(ctx, log, a, clusterRG, testCRGName, tt.zones, tt.topology)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
