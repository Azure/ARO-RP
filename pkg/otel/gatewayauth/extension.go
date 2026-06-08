package gatewayauth

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/otel/gatewayauth/internal/metadata"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

type gatewayAuthenticationExtension struct {
	id component.ID

	_env          env.Core
	serverAuthLog *logrus.Entry

	// lifetime context (e.g. for refresh loops)
	ctx    context.Context
	cancel context.CancelCauseFunc

	cfg    *Config
	params extension.Settings

	auth *authManager
}

var _ extensionauth.Server = &gatewayAuthenticationExtension{}

// Start is called by the Collector when the extension is started.
func (e *gatewayAuthenticationExtension) Start(_ctx context.Context, _ component.Host) error {
	if e.ctx != nil {
		return errors.New("already started")
	}

	// Make a new cancellable background context for ongoing tasks as _ctx might
	// only be valid during the startup phase
	ctx, cancel := context.WithCancelCause(context.Background())
	e.ctx = ctx
	e.cancel = cancel

	tlscfg := e.cfg.TLS.Get()
	if tlscfg == nil {
		err := errors.New("tls config is required")
		cancel(err)
		e.ctx = nil
		e.cancel = nil
		return err
	}
	got, err := tlscfg.LoadTLSConfig(ctx)
	if err != nil {
		cancel(err)
		e.ctx = nil
		e.cancel = nil
		return err
	}
	tls := credentials.NewTLS(got)

	err = e.startChangefeed(ctx, e.params.Logger, tls)
	if err != nil {
		cancel(err)
		e.ctx = nil
		e.cancel = nil
		return err
	}

	e.params.Logger.Info("extension started", zap.String("id", e.id.String()))
	return nil
}

// Set up the Gateway changefeed cache
func (e *gatewayAuthenticationExtension) startChangefeed(ctx context.Context, log *zap.Logger, tls credentials.TransportCredentials) error {
	_env, err := env.NewCore(ctx, utillog.NewLogrusToZapLogger(log), env.SERVICE_LOG_COLLECTOR)
	if err != nil {
		return err
	}

	m := statsd.New(ctx, _env, os.Getenv("MDM_ACCOUNT"), os.Getenv("MDM_NAMESPACE"), os.Getenv("MDM_STATSD_SOCKET"))
	go m.Run(ctx.Done())

	g, err := golang.NewMetrics(_env.LoggerForComponent("metrics"), m)
	if err != nil {
		return err
	}
	go g.Run()

	// don't need the AEAD for the gateway database
	dbc, err := database.NewDatabaseClientFromEnv(ctx, _env, m, nil)
	if err != nil {
		return err
	}

	dbName, err := env.DBName(_env)
	if err != nil {
		return err
	}

	dbGateway, err := database.NewGateway(ctx, dbc, dbName)
	if err != nil {
		return err
	}

	interv := e.cfg.ChangefeedInterval.Get()
	if interv == nil {
		interv = pointerutils.ToPtr(30)
	}

	batchSize := e.cfg.ChangefeedBatchSize.Get()
	if batchSize == nil {
		batchSize = pointerutils.ToPtr(500)
	}

	e._env = _env
	e.serverAuthLog = _env.LoggerForComponent("serverAuthenticator")
	e.auth = newAuthManager(_env, tls, m, time.Second*time.Duration(*interv), *batchSize)

	// start the changefeed to refresh the gateway linkID cache
	go e.auth.startChangefeed(ctx, dbGateway)

	return nil
}

// Shutdown is called by the Collector when the extension is stopped.
func (e *gatewayAuthenticationExtension) Shutdown(_ context.Context) error {
	if e.cancel != nil {
		e.cancel(errors.New("stop() called"))
	}
	e.ctx = nil
	e.cancel = nil
	e.auth = nil
	e._env = nil
	e.serverAuthLog = nil

	e.params.Logger.Info("extension stopped", zap.String("id", e.id.String()))
	return nil
}

func (e *gatewayAuthenticationExtension) GetGRPCServerOptions(_ context.Context) ([]grpc.ServerOption, error) {
	if e.auth == nil {
		return nil, errors.New("not started")
	}

	return []grpc.ServerOption{grpc.Creds(e.auth)}, nil
}

func createGatewayAuthenticationExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	cfg2 := cfg.(*Config)
	return &gatewayAuthenticationExtension{id: set.ID, params: set, cfg: cfg2}, nil
}

func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createAuthConfig,
		createGatewayAuthenticationExtension,
		component.StabilityLevelDevelopment,
	)
}
