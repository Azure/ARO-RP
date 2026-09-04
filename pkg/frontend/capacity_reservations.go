package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

type crgActions interface {
	CreateCRG(ctx context.Context, clusterRG, location string, zones []string, crgName string) (string, error)
	CreateCapacityReservation(ctx context.Context, clusterRG, location, zone, targetSKU, crgName string, capacity int64) error
	DeleteCRG(ctx context.Context, clusterRG, crgName string) error
	DeleteCapacityReservation(ctx context.Context, clusterRG, crgName, zone string) error
}

// Returns true even on error if the CRG was created — caller must run crgTeardown regardless.
func crgSetupForResize(ctx context.Context, log *logrus.Entry, a crgActions, clusterRG, location string, zones []string, topology zoneTopology, targetSKU, crgName string) (bool, error) {
	log.Infof("setting up capacity reservation group %s for SKU %s", crgName, targetSKU)

	var crgZones []string
	if topology == zoneTopologyThreeZone {
		crgZones = zones
	}
	if _, err := a.CreateCRG(ctx, clusterRG, location, crgZones, crgName); err != nil {
		return false, fmt.Errorf("creating capacity reservation group: %w", err)
	}

	switch topology {
	case zoneTopologyThreeZone:
		for _, zone := range zones {
			if err := a.CreateCapacityReservation(ctx, clusterRG, location, zone, targetSKU, crgName, 1); err != nil {
				return true, fmt.Errorf("creating capacity reservation in zone %s: %w", zone, err)
			}
		}
	case zoneTopologyRegional:
		if err := a.CreateCapacityReservation(ctx, clusterRG, location, "", targetSKU, crgName, 3); err != nil {
			return true, fmt.Errorf("creating regional capacity reservation: %w", err)
		}
	}

	log.Infof("capacity reservation group %s ready for resize to %s", crgName, targetSKU)
	return true, nil
}

func crgTeardown(ctx context.Context, log *logrus.Entry, a crgActions, clusterRG, crgName string, zones []string, topology zoneTopology) error {
	log.Infof("tearing down capacity reservation group %s", crgName)

	var errs []error
	switch topology {
	case zoneTopologyThreeZone:
		for _, zone := range zones {
			if err := a.DeleteCapacityReservation(ctx, clusterRG, crgName, zone); err != nil {
				errs = append(errs, err)
			}
		}
	case zoneTopologyRegional:
		if err := a.DeleteCapacityReservation(ctx, clusterRG, crgName, ""); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return a.DeleteCRG(ctx, clusterRG, crgName)
}
