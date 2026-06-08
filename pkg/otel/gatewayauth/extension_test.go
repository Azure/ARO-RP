package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensiontest"
)

func TestStartReturnsErrorWhenAlreadyStarted(t *testing.T) {
	e := &gatewayAuthenticationExtension{
		ctx: context.Background(),
	}

	err := e.Start(context.Background(), nil)
	require.ErrorContains(t, err, "already started")
}

func TestGetGRPCServerOptionsReturnsErrorWhenNotStarted(t *testing.T) {
	e := &gatewayAuthenticationExtension{}

	_, err := e.GetGRPCServerOptions(context.Background())
	require.ErrorContains(t, err, "not started")
}

func TestShutdownClearsRuntimeState(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	e := &gatewayAuthenticationExtension{
		ctx:    ctx,
		cancel: cancel,
		auth:   &authManager{},
		params: extensiontest.NewNopSettings(component.MustNewType("gatewayauth")),
	}

	err := e.Shutdown(context.Background())
	require.NoError(t, err)
	require.Nil(t, e.ctx)
	require.Nil(t, e.cancel)
	require.Nil(t, e.auth)
	require.Nil(t, e._env)
	require.Nil(t, e.serverAuthLog)
}
