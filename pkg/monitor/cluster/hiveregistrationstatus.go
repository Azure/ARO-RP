package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

type hiveRegistrationState struct {
	isRegistered        bool
	isUnreachable       bool
	notRegisteredReason string
	notReachableReason  string
}

func (mon *Monitor) emitHiveRegistrationStatus(ctx context.Context) error {
	if mon.hiveClusterManager == nil {
		return nil //nothing to do
	}

	mon.log.Info(mon.oc.Name)
	state, err := mon.hiveRegistrationState(ctx)
	if err != nil {
		return err
	}
	mon.log.Info(state)

	if !state.isRegistered {
		mon.emitGauge("hive.registration.error", 1, map[string]string{
			"reason": state.notRegisteredReason,
		})
	}

	if state.isUnreachable {
		mon.emitGauge("hive.registration.isUnreachable", 1, map[string]string{
			"reason": state.notReachableReason,
		})
	}

	return nil
}

func (mon *Monitor) hiveRegistrationState(ctx context.Context) (hiveRegistrationState, error) {
	state := hiveRegistrationState{ //default to all is good
		isRegistered:        true,
		isUnreachable:       false,
		notRegisteredReason: "",
		notReachableReason:  "",
	}

	if mon.oc.Properties.HiveProfile.Namespace == "" {
		state.notRegisteredReason = "no namespace in cluster document"
		return state, nil
	}

	isConnected, reason, err := mon.hiveClusterManager.IsConnected(ctx, mon.oc.Properties.HiveProfile.Namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			state.isRegistered = false
			state.isUnreachable = true
			state.notRegisteredReason = "namespace or clusterdeployment not found"
			state.notReachableReason = "cluster not registered"
			return state, nil
		}
		return state, err
	}

	if !isConnected {
		state.isRegistered = true
		state.isUnreachable = true
		state.notRegisteredReason = ""
		state.notRegisteredReason = reason
		return state, nil
	}

	return state, nil
}
