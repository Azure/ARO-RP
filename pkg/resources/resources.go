package resources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/resources/vnet"
)

// Resource represents default interface for the individual resource providers
type Resource interface {
	// Allocate allocated individual resource based on atomic operation.
	Allocate() (interface{}, error)

	// Deallocate releases (expires) individual resource, leaving it for managed
	// to deal with.
	Deallocate(resource interface{}) error
}

// ResourceManager respresents resource manager interface for all providers
type ResourceManager interface {
	// Allocate allocated individual resource based on atomic operation.
	Allocate(resourceType api.ResourceType) (interface{}, error)

	// Deallocate releases (expires) individual resource, leaving it for managed
	// to deal with.
	Deallocate(resourceType api.ResourceType, resource interface{}) error
}

type resourceManager struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database
	m       metrics.Interface

	// individual resource providers
	vnetRP Resource
}

func New(ctx context.Context, log *logrus.Entry, env env.Interface, db *database.Database, m metrics.Interface) (*resourceManager, error) {
	// instantiate individual resource providers
	vnetRP, err := vnet.New(ctx, log, env, db, m)
	if err != nil {
		return nil, err
	}

	return &resourceManager{
		baseLog: log,
		env:     env,
		db:      db,
		m:       m,

		vnetRP: vnetRP,
	}, nil
}

func (m *resourceManager) Allocate(resourceType api.ResourceType) (interface{}, error) {
	switch resourceType {
	case api.ResourceTypePEVNET:
		return m.vnetRP.Allocate()
	default:
		return nil, fmt.Errorf("unknown resource provider %s", resourceType)
	}
}

func (m *resourceManager) Deallocate(resourceType api.ResourceType, resource interface{}) error {
	switch resourceType {
	case api.ResourceTypePEVNET:
		return m.vnetRP.Deallocate(resource)
	default:
		return fmt.Errorf("unknown resource provider %s", resourceType)
	}
}
