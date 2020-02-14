package vnet

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/virtualnetwork"
)

// vnetThreshhold defines threshold, when manager should create new vnet and
// pair it with main vnet for resource pool extension
var vnetThreshhold = 10

// vnetResourceManager is responsible for tracking how many free IP addresses
// there are in the individual vnet. Because PrivateEndpoint does not allow
// to provide static IP address, we will be tracking just total amount of IPs
// available and store attached cluster data into individual resource definition

type vnetResourceManager struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database
	m       metrics.Interface
}

func New(ctx context.Context, log *logrus.Entry, env env.Interface, db *database.Database, metrics metrics.Interface) (*vnetResourceManager, error) {
	return &vnetResourceManager{
		baseLog: log,
		env:     env,
		db:      db,
		m:       metrics,
	}, nil
}

func (m *vnetResourceManager) run(ctx context.Context, log *logrus.Entry, env env.Interface, db database.Database, metrics metrics.Interface) error {
	localFPAuthorizer, err := env.FPAuthorizer(env.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return err
	}

	// TODO:
	// 1. Set subnet address namespace step
	// 2. Create vnet name map with subnets address space
	// 3. Check which of those subnets exists in RP
	// 4. Check document store to see if we have same view
	// 5. Update document store with corresponding view of the real world.
	virtualnetwork.NewManager(env, localFPAuthorizer)

	return nil
}

func (m *vnetResourceManager) Allocate() (interface{}, error) {
	// TODO:
	// Take first vnet in the list with free space and allocate to the cluster.
	// TODO: Cluster should report if it used this resource somehow
	return nil, nil
}

func (m *vnetResourceManager) Deallocate(resource interface{}) error {
	// TODO:
	// Not sure if this is needed for vnets. If we not gonna track individual
	// IP's cleaning clusters should delete PE and release resources. Manager should
	// pick those up and re-distribute.
	return nil
}
