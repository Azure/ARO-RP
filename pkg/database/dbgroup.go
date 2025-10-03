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

type DatabaseGroupWithOpenShiftVersions interface {
	OpenShiftVersions() (OpenShiftVersions, error)
}

type DatabaseGroupWithPlatformWorkloadIdentityRoleSets interface {
	PlatformWorkloadIdentityRoleSets() (PlatformWorkloadIdentityRoleSets, error)
}

type DatabaseGroupWithAsyncOperations interface {
	AsyncOperations() (AsyncOperations, error)
}

type DatabaseGroupWithBilling interface {
	Billing() (Billing, error)
}

type DatabaseGroupWithPortal interface {
	Portal() (Portal, error)
}

type DatabaseGroupWithMaintenanceManifests interface {
	MaintenanceManifests() (MaintenanceManifests, error)
}

type DatabaseGroup interface {
	DatabaseGroupWithOpenShiftClusters
	DatabaseGroupWithSubscriptions
	DatabaseGroupWithMonitors
	DatabaseGroupWithOpenShiftVersions
	DatabaseGroupWithPlatformWorkloadIdentityRoleSets
	DatabaseGroupWithAsyncOperations
	DatabaseGroupWithBilling
	DatabaseGroupWithPortal
	DatabaseGroupWithMaintenanceManifests

	WithOpenShiftClusters(db OpenShiftClusters) DatabaseGroup
	WithSubscriptions(db Subscriptions) DatabaseGroup
	WithMonitors(db Monitors) DatabaseGroup
	WithOpenShiftVersions(db OpenShiftVersions) DatabaseGroup
	WithPlatformWorkloadIdentityRoleSets(db PlatformWorkloadIdentityRoleSets) DatabaseGroup
	WithAsyncOperations(db AsyncOperations) DatabaseGroup
	WithBilling(db Billing) DatabaseGroup
	WithPortal(db Portal) DatabaseGroup
	WithMaintenanceManifests(db MaintenanceManifests) DatabaseGroup
}

type dbGroup struct {
	openShiftClusters                OpenShiftClusters
	subscriptions                    Subscriptions
	monitors                         Monitors
	platformWorkloadIdentityRoleSets PlatformWorkloadIdentityRoleSets
	openShiftVersions                OpenShiftVersions
	asyncOperations                  AsyncOperations
	billing                          Billing
	portal                           Portal
	maintenanceManifests             MaintenanceManifests
}

func (d *dbGroup) OpenShiftClusters() (OpenShiftClusters, error) {
	if d.openShiftClusters == nil {
		return nil, errors.New("no OpenShiftClusters database client set")
	}
	return d.openShiftClusters, nil
}

func (d *dbGroup) WithOpenShiftClusters(db OpenShiftClusters) DatabaseGroup {
	d.openShiftClusters = db
	return d
}

func (d *dbGroup) Subscriptions() (Subscriptions, error) {
	if d.subscriptions == nil {
		return nil, errors.New("no Subscriptions database client set")
	}
	return d.subscriptions, nil
}

func (d *dbGroup) WithSubscriptions(db Subscriptions) DatabaseGroup {
	d.subscriptions = db
	return d
}

func (d *dbGroup) Monitors() (Monitors, error) {
	if d.monitors == nil {
		return nil, errors.New("no Monitors database client set")
	}
	return d.monitors, nil
}

func (d *dbGroup) WithMonitors(db Monitors) DatabaseGroup {
	d.monitors = db
	return d
}

func (d *dbGroup) OpenShiftVersions() (OpenShiftVersions, error) {
	if d.openShiftVersions == nil {
		return nil, errors.New("no OpenShiftVersions database client set")
	}
	return d.openShiftVersions, nil
}

func (d *dbGroup) WithOpenShiftVersions(db OpenShiftVersions) DatabaseGroup {
	d.openShiftVersions = db
	return d
}

func (d *dbGroup) PlatformWorkloadIdentityRoleSets() (PlatformWorkloadIdentityRoleSets, error) {
	if d.platformWorkloadIdentityRoleSets == nil {
		return nil, errors.New("no PlatformWorkloadIdentityRoleSets database client set")
	}
	return d.platformWorkloadIdentityRoleSets, nil
}

func (d *dbGroup) WithPlatformWorkloadIdentityRoleSets(db PlatformWorkloadIdentityRoleSets) DatabaseGroup {
	d.platformWorkloadIdentityRoleSets = db
	return d
}

func (d *dbGroup) AsyncOperations() (AsyncOperations, error) {
	if d.asyncOperations == nil {
		return nil, errors.New("no AsyncOperations database client set")
	}
	return d.asyncOperations, nil
}

func (d *dbGroup) WithAsyncOperations(db AsyncOperations) DatabaseGroup {
	d.asyncOperations = db
	return d
}

func (d *dbGroup) Billing() (Billing, error) {
	if d.billing == nil {
		return nil, errors.New("no Billing database client set")
	}
	return d.billing, nil
}

func (d *dbGroup) WithBilling(db Billing) DatabaseGroup {
	d.billing = db
	return d
}

func (d *dbGroup) Portal() (Portal, error) {
	if d.portal == nil {
		return nil, errors.New("no Portal database client set")
	}
	return d.portal, nil
}

func (d *dbGroup) WithPortal(db Portal) DatabaseGroup {
	d.portal = db
	return d
}

func (d *dbGroup) MaintenanceManifests() (MaintenanceManifests, error) {
	if d.maintenanceManifests == nil {
		return nil, errors.New("no MaintenanceManifests database client set")
	}
	return d.maintenanceManifests, nil
}

func (d *dbGroup) WithMaintenanceManifests(db MaintenanceManifests) DatabaseGroup {
	d.maintenanceManifests = db
	return d
}

func NewDBGroup() DatabaseGroup {
	return &dbGroup{}
}
