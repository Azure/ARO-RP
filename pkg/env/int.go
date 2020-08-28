package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

func newInt(ctx context.Context, log *logrus.Entry, instancemetadata instancemetadata.InstanceMetadata) (*prod, error) {
	p, err := newProd(ctx, log, instancemetadata)

	if err != nil {
		return nil, err
	}

	p.fpServicePrincipalID = "71cfb175-ea3a-444e-8c03-b119b2752ce4"
	p.clustersGenevaLoggingEnvironment = "Test"
	p.clustersGenevaLoggingConfigVersion = "2.2"
	p.envType = Int

	return p, nil
}
