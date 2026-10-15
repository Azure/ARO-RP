package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"time"

	sdkazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

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
