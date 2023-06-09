package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

type ContainerInstaller interface {
	Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error
}

type manager struct {
	conn context.Context
	log  *logrus.Entry
	env  env.Interface

	doc     *api.OpenShiftClusterDocument
	sub     *api.SubscriptionDocument
	version *api.OpenShiftVersion

	isDevelopment bool

	success bool
}

func New(ctx context.Context, log *logrus.Entry, env env.Interface) (ContainerInstaller, error) {
	isDevelopment := env.IsLocalDevelopmentMode()
	conn, err := getConnection(ctx, isDevelopment)
	if err != nil {
		return nil, err
	}

	return &manager{
		conn:          conn,
		log:           log,
		env:           env,
		isDevelopment: isDevelopment,
	}, nil
}
