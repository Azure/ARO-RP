package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"time"

	sdkazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type Rebuilder interface {
	Rebuild() error
}

type Authorizer interface {
	autorest.Authorizer
	Rebuild() error
}

type TokenCredential interface {
	azcore.TokenCredential
	Rebuilder
}

type tokenCredential struct {
	env               env.Interface
	tenantID          string
	additionalTenants []string

	m    sync.RWMutex
	cred azcore.TokenCredential
}

func (c *tokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (sdkazcore.AccessToken, error) {
	c.m.RLock()
	cred := c.cred
	c.m.RUnlock()

	tk, err := cred.GetToken(ctx, options)
	if err != nil {
		return tk, err
	}

	tk.ExpiresOn = time.Time{}

	return tk, nil
}

func (c *tokenCredential) Rebuild() error {
	cred, err := c.env.FPNewClientCertificateCredential(c.tenantID, c.additionalTenants)
	if err != nil {
		return err
	}

	c.m.Lock()
	c.cred = cred
	c.m.Unlock()

	return nil
}

func NewFPTokenCredential(_env env.Interface, tenantID string, additionalTenants []string) (TokenCredential, error) {
	c := &tokenCredential{
		env:               _env,
		tenantID:          tenantID,
		additionalTenants: additionalTenants,
	}

	err := c.Rebuild()
	if err != nil {
		return nil, err
	}

	return c, nil
}

type multiRebuilder []Rebuilder

func (m multiRebuilder) Rebuild() error {
	for _, r := range m {
		if r == nil {
			continue
		}

		err := r.Rebuild()
		if err != nil {
			return err
		}
	}

	return nil
}

func NewMultiRebuilder(rebuilders ...Rebuilder) Rebuilder {
	return multiRebuilder(rebuilders)
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
