package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/aad"
)

func (mon *Monitor) emitTokenVerify(ctx context.Context) error {
	if !mon.hourlyRun {
		return nil
	}

	_, err := aad.GetToken(ctx, mon.log, mon.oc, azure.PublicCloud.GraphEndpoint, false)
	if err != nil {
		mon.emitGauge("cluster.auth.errors", 1, map[string]string{})
		mon.log.WithFields(logrus.Fields{
			"metric": "cluster.auth.errors",
			"error":  err,
		}).Print()
	}

	return nil
}
