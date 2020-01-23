package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/env"
	basekeyvault "github.com/Azure/ARO-RP/pkg/util/azureclient/keyvault"
)

type Manager interface {
}

type manager struct {
	env      env.Interface
	keyvault basekeyvault.BaseClient
}

func NewManager(env env.Interface, localFPKVAuthorizer autorest.Authorizer) Manager {
	return &manager{
		env: env,

		keyvault: basekeyvault.New(localFPKVAuthorizer),
	}
}
