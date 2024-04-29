package containerinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

type ContainerInstaller interface {
	Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error
}

type manager struct {
	conn context.Context
	log  *logrus.Entry
	env  env.Interface

	clusterUUID string
	pullSecret  *pullsecret.UserPass

	success bool
}

func New(ctx context.Context, log *logrus.Entry, env env.Interface, clusterUUID string) (ContainerInstaller, error) {
	isDevelopment := env.IsLocalDevelopmentMode()
	if !isDevelopment {
		return nil, errors.New("running cluster installs in a container is only run in development")
	}

	pullSecret, err := pullsecret.Extract(env.GetEnv("PULL_SECRET"), env.ACRDomain())
	if err != nil {
		return nil, err
	}

	conn, err := getConnection(ctx, env)
	if err != nil {
		return nil, err
	}

	return &manager{
		conn: conn,
		log:  log,
		env:  env,

		clusterUUID: clusterUUID,
		pullSecret:  pullSecret,
	}, nil
}
