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

	p.fpServicePrincipalID = "71cfb175-ea3a-444e-8c03-b119b2752ce4"
	p.clustersGenevaLoggingEnvironment = "Test"
	p.clustersGenevaLoggingConfigVersion = "2.2"
	p.e2eStorageAccountName = "arov4e2eint"
	p.e2eStorageAccountRGName = "global-infra"
	p.e2eStorageAccountSubID = "0cc1cafa-578f-4fa5-8d6b-ddfd8d82e6ea"

	return p, nil
}
