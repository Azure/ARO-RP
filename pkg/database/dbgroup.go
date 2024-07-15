package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "errors"

type DatabaseGroupWithOpenShiftClusters interface {
	OpenShiftClusters() (OpenShiftClusters, error)
}

type DatabaseGroupWithSubscriptions interface {
	Subscriptions() (Subscriptions, error)
}

type DatabaseGroupWithMonitors interface {
	Monitors() (Monitors, error)
}

type DBGroup interface {
	DatabaseGroupWithOpenShiftClusters
	DatabaseGroupWithSubscriptions
	DatabaseGroupWithMonitors
}

type dbGroup struct {
	openShiftClusters OpenShiftClusters
	subscriptions     Subscriptions
	monitors          Monitors
}

func (d *dbGroup) OpenShiftClusters() (OpenShiftClusters, error) {
	if d.openShiftClusters == nil {
		return nil, errors.New("no OpenShiftClusters defined")
	}
	return d.openShiftClusters, nil
}

func (d *dbGroup) WithOpenShiftClusters(o OpenShiftClusters) *dbGroup {
	d.openShiftClusters = o
	return d
}

func (d *dbGroup) Subscriptions() (Subscriptions, error) {
	if d.subscriptions == nil {
		return nil, errors.New("no Subscriptions defined")
	}
	return d.subscriptions, nil
}

func (d *dbGroup) WithSubscriptions(s Subscriptions) *dbGroup {
	d.subscriptions = s
	return d
}
func (d *dbGroup) Monitors() (Monitors, error) {
	if d.monitors == nil {
		return nil, errors.New("no Monitors defined")
	}
	return d.monitors, nil
}

func (d *dbGroup) WithMonitors(m Monitors) *dbGroup {
	d.monitors = m
	return d
}

func NewDBGroup() *dbGroup {
	return &dbGroup{}
}
