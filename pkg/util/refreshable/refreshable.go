package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
)

type Authorizer interface {
	autorest.Authorizer
	RefreshWithContext(ctx context.Context) error
}

type authorizer struct {
	autorest.Authorizer
	sp *adal.ServicePrincipalToken
}

func (a *authorizer) RefreshWithContext(ctx context.Context) error {
	return a.sp.RefreshWithContext(ctx)
}

func NewAuthorizer(sp *adal.ServicePrincipalToken) Authorizer {
	return &authorizer{
		Authorizer: autorest.NewBearerAuthorizer(sp),
		sp:         sp,
	}
}
