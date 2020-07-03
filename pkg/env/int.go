package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

func newInt(ctx context.Context, log *logrus.Entry, instancemetadata instancemetadata.InstanceMetadata) (*prod, error) {
	kvAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	rpAuthorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	p, err := newProd(ctx, log, instancemetadata, rpAuthorizer, kvAuthorizer)

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
