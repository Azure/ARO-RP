package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/env"
)

type Authorizer interface {
	autorest.Authorizer
	Rebuild() error
}

type authorizer struct {
	auth     autorest.Authorizer
	env      env.Interface
	tenantID string
}

func (a *authorizer) Rebuild() error {
	auth, err := a.env.FPAuthorizer(a.tenantID, nil, a.env.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}
	a.auth = auth
	return nil
}

func (a *authorizer) WithAuthorization() autorest.PrepareDecorator {
	return a.auth.WithAuthorization()
}

// NewAuthorizer creates an Authorizer that can be rebuilt when needed to force
// token recreation.
func NewAuthorizer(_env env.Interface, tenantID string) (Authorizer, error) {
	a := &authorizer{
		env:      _env,
		tenantID: tenantID,
	}
	err := a.Rebuild()
	return a, err
}
