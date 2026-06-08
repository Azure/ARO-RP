package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/client"
	"google.golang.org/grpc/peer"
)

var (
	ErrNotNegotiatedByUs   = errors.New("connection was not negotiated by us (wrong AuthInfo)")
	ErrUnableToGetPeer     = errors.New("unable to get peer from context")
	ErrUnableToGetAuthInfo = errors.New("unable to get authinfo from peer")
)

func (e *gatewayAuthenticationExtension) Authenticate(ctx context.Context, sources map[string][]string) (context.Context, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, ErrUnableToGetPeer
	}

	if p == nil || p.AuthInfo == nil {
		return ctx, ErrUnableToGetAuthInfo
	}

	gatewayAuthInfo, ok := p.AuthInfo.(gatewayAuthInfo)
	if !ok {
		e.serverAuthLog.Error(ErrNotNegotiatedByUs)
		return nil, ErrNotNegotiatedByUs
	}

	cl := client.FromContext(ctx)
	cl.Auth = gatewayAuthInfo
	return client.NewContext(ctx, cl), nil
}
