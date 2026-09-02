package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

const testCRGName = "aro-resize-crg-cp-test"

type fakeCRGActions struct {
	createCRGFn                 func(ctx context.Context, clusterRG, location string, zones []string, crgName string) (string, error)
	createCapacityReservationFn func(ctx context.Context, clusterRG, location, zone, targetSKU, crgName string, capacity int64) error
	deleteCRGFn                 func(ctx context.Context, clusterRG, crgName string) error
	deleteCapacityReservationFn func(ctx context.Context, clusterRG, crgName, zone string) error
}

func (f *fakeCRGActions) CreateCRG(ctx context.Context, clusterRG, location string, zones []string, crgName string) (string, error) {
	return f.createCRGFn(ctx, clusterRG, location, zones, crgName)
}

func (f *fakeCRGActions) CreateCapacityReservation(ctx context.Context, clusterRG, location, zone, targetSKU, crgName string, capacity int64) error {
	return f.createCapacityReservationFn(ctx, clusterRG, location, zone, targetSKU, crgName, capacity)
}

func (f *fakeCRGActions) DeleteCRG(ctx context.Context, clusterRG, crgName string) error {
	return f.deleteCRGFn(ctx, clusterRG, crgName)
}

func (f *fakeCRGActions) DeleteCapacityReservation(ctx context.Context, clusterRG, crgName, zone string) error {
	return f.deleteCapacityReservationFn(ctx, clusterRG, crgName, zone)
}

func TestCRGSetupForResize(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())
	crgID := "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/" + testCRGName

	tests := []struct {
		name        string
		fake        *fakeCRGActions
		wantCreated bool
		wantErr     string
	}{
		{
			name: "happy path",
			fake: &fakeCRGActions{
				createCRGFn: func(_ context.Context, _, _ string, _ []string, _ string) (string, error) {
					return crgID, nil
				},
				createCapacityReservationFn: func(_ context.Context, _, _, _, _, _ string, _ int64) error {
					return nil
				},
			},
			wantCreated: true,
		},
		{
			name: "CRG create fails",
			fake: &fakeCRGActions{
				createCRGFn: func(_ context.Context, _, _ string, _ []string, _ string) (string, error) {
					return "", errors.New("network error")
				},
			},
			wantCreated: false,
			wantErr:     "creating capacity reservation group: network error",
		},
		{
			name: "reservation fails with capacity error - returns true so caller can teardown",
			fake: &fakeCRGActions{
				createCRGFn: func(_ context.Context, _, _ string, _ []string, _ string) (string, error) {
					return crgID, nil
				},
				createCapacityReservationFn: func(_ context.Context, _, _, _, _, _ string, _ int64) error {
					return &azcore.ResponseError{ErrorCode: "AllocationFailed", StatusCode: http.StatusConflict}
				},
			},
			wantCreated: true,
			wantErr:     "creating capacity reservation in zone 1: " + (&azcore.ResponseError{ErrorCode: "AllocationFailed", StatusCode: http.StatusConflict}).Error(),
		},
		{
			name: "stops at first reservation failure",
			fake: &fakeCRGActions{
				createCRGFn: func(_ context.Context, _, _ string, _ []string, _ string) (string, error) {
					return crgID, nil
				},
				createCapacityReservationFn: func(_ context.Context, _, _, zone, _, _ string, _ int64) error {
					if zone == "1" {
						return errors.New("zone 1 failed")
					}
					t.Errorf("unexpected call for zone %s", zone)
					return nil
				},
			},
			wantCreated: true,
			wantErr:     "creating capacity reservation in zone 1: zone 1 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crgCreated, err := crgSetupForResize(ctx, log, tt.fake, "cluster-rg", "eastus", []string{"1", "2"}, "Standard_D16s_v3", testCRGName)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if crgCreated != tt.wantCreated {
				t.Errorf("crgCreated = %v, want %v", crgCreated, tt.wantCreated)
			}
		})
	}
}

func TestCRGTeardown(t *testing.T) {
	ctx := context.Background()
	log := logrus.NewEntry(logrus.StandardLogger())

	tests := []struct {
		name    string
		fake    *fakeCRGActions
		wantErr string
	}{
		{
			name: "happy path",
			fake: &fakeCRGActions{
				deleteCapacityReservationFn: func(_ context.Context, _, _, _ string) error { return nil },
				deleteCRGFn:                 func(_ context.Context, _, _ string) error { return nil },
			},
		},
		{
			name: "reservation delete fails - skips CRG delete",
			fake: &fakeCRGActions{
				deleteCapacityReservationFn: func(_ context.Context, _, _, zone string) error {
					if zone == "1" {
						return errors.New("zone 1 failed")
					}
					return nil
				},
				deleteCRGFn: func(_ context.Context, _, _ string) error {
					t.Error("DeleteCRG should not be called when reservations fail")
					return nil
				},
			},
			wantErr: "zone 1 failed",
		},
		{
			name: "CRG delete fails after all reservations removed",
			fake: &fakeCRGActions{
				deleteCapacityReservationFn: func(_ context.Context, _, _, _ string) error { return nil },
				deleteCRGFn:                 func(_ context.Context, _, _ string) error { return errors.New("CRG delete failed") },
			},
			wantErr: "CRG delete failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := crgTeardown(ctx, log, tt.fake, "cluster-rg", testCRGName, []string{"1", "2"})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
