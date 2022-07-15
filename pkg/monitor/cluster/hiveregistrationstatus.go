package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

type hiveRegistrationState struct {
	isReachable        bool
	notReachableReason string
}

func (mon *Monitor) emitHiveRegistrationStatus(ctx context.Context) error {
	if mon.hiveClusterManager == nil {
		return nil
	}

	state, err := mon.hiveRegistrationState(ctx)
	if err != nil {
		return err
	}

	if !state.isReachable {
		mon.emitGauge("hive.registration.isUnreachable", 1, map[string]string{
			"reason": state.notReachableReason,
		})
	}

	return nil
}

func (mon *Monitor) hiveRegistrationState(ctx context.Context) (hiveRegistrationState, error) {
	state := hiveRegistrationState{
		isReachable:        false,
		notReachableReason: "",
	}

	if mon.oc.Properties.HiveProfile.Namespace == "" {
		state.notReachableReason = "no namespace in cluster document"
		return state, nil
	}

	isConnected, reason, err := mon.hiveClusterManager.IsConnected(ctx, mon.oc.Properties.HiveProfile.Namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			state.notReachableReason = "cluster not registered"
			return state, nil
		}
		return state, err
	}

	state.isReachable = isConnected
	state.notReachableReason = reason

	return state, nil
}
