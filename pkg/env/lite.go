package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

type Lite interface {
	instancemetadata.InstanceMetadata

	Type() Type
}

type lite struct {
	instancemetadata.InstanceMetadata
}

func (*lite) Type() Type {
	return envType()
}

func NewEnvLite(ctx context.Context, log *logrus.Entry) (Lite, error) {
	if envType() == Dev {
		log.Warn("running in development mode")
	}

	im, err := newInstanceMetadata(ctx)
	if err != nil {
		return nil, err
	}

	return &lite{InstanceMetadata: im}, nil
}
