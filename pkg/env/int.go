package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
)

func newInt(ctx context.Context, log *logrus.Entry) (*prod, error) {
	p, err := newProd(ctx, log)

	if err != nil {
		return nil, err
	}

	p.fpClientID = "71cfb175-ea3a-444e-8c03-b119b2752ce4"
	p.clusterGenevaLoggingEnvironment = "Test"
	p.clusterGenevaLoggingConfigVersion = "2.2"

	return p, nil
}
