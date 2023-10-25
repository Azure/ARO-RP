package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd"
	"github.com/Azure/ARO-RP/pkg/metrics/statsd/golang"
	"github.com/Azure/ARO-RP/pkg/mimo/actuator"
	"github.com/Azure/ARO-RP/pkg/proxy"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/service"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log := utillog.GetLogger()
	developmentMode := env.IsLocalDevelopmentMode()

	app := &cli.App{
		Name:  "MIMO",
		Usage: "Managed Infrastructure Maintenance Operator",
		Commands: []*cli.Command{
			{
				Name: "scheduler",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "MDM_ACCOUNT",
						EnvVars:  []string{"MDM_ACCOUNT"},
						Required: !developmentMode,
					},
					&cli.StringFlag{
						Name:     "MDM_NAMESPACE",
						EnvVars:  []string{"MDM_NAMESPACE"},
						Required: !developmentMode,
					},
					&cli.StringFlag{
						Name:     "MDM_STATSD_SOCKET",
						EnvVars:  []string{"MDM_STATSD_SOCKET"},
						Required: false,
					},
				},
				Action: func(ctx *cli.Context) error {
					_env, err := env.NewEnv(ctx.Context, log, env.COMPONENT_MIMO_SCHEDULER)
					if err != nil {
						return err
					}

					m := statsd.NewFromEnv(ctx.Context, _env.Logger(), _env)

					g, err := golang.NewMetrics(_env.Logger(), m)
					if err != nil {
						return err
					}
					go g.Run()

					// dbc, err := service.NewDatabase(ctx.Context, _env, log, m, service.DB_DBTOKEN_PROD_MASTERKEY_DEV, false)
					// if err != nil {
					// 	return err
					// }

					// dbName, err := service.DBName(_env.IsLocalDevelopmentMode())
					// if err != nil {
					// 	return err
					// }

					// clusters, err := database.NewOpenShiftClusters(ctx.Context, dbc, dbName)
					// if err != nil {
					// 	return err
					// }

					return nil
				},
			},
			{
				Name: "actuator",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "MDM_ACCOUNT",
						EnvVars:  []string{"MDM_ACCOUNT"},
						Required: !developmentMode,
					},
					&cli.StringFlag{
						Name:     "MDM_NAMESPACE",
						EnvVars:  []string{"MDM_NAMESPACE"},
						Required: !developmentMode,
					},
					&cli.StringFlag{
						Name:     "MDM_STATSD_SOCKET",
						EnvVars:  []string{"MDM_STATSD_SOCKET"},
						Required: false,
					},
				},
				Before: func(ctx *cli.Context) error {
					log.Print("MIMO actuator initialising")
					return nil
				},
				Action: func(ctx *cli.Context) error {
					stop := make(chan struct{})

					_env, err := env.NewEnv(ctx.Context, log, env.COMPONENT_MIMO_ACTUATOR)
					if err != nil {
						return err
					}

					m := statsd.NewFromEnv(ctx.Context, _env.Logger(), _env)

					g, err := golang.NewMetrics(_env.Logger(), m)
					if err != nil {
						return err
					}
					go g.Run()

					dbc, err := service.NewDatabase(ctx.Context, _env, log, m, service.DB_DBTOKEN_PROD_MASTERKEY_DEV, false)
					if err != nil {
						return err
					}

					dbName, err := service.DBName(_env.IsLocalDevelopmentMode())
					if err != nil {
						return err
					}

					buckets, err := database.NewBucketServices(ctx.Context, dbc, dbName)
					if err != nil {
						return err
					}

					clusters, err := database.NewOpenShiftClusters(ctx.Context, dbc, dbName)
					if err != nil {
						return err
					}

					manifests, err := database.NewMaintenanceManifests(ctx.Context, dbc, dbName)
					if err != nil {
						return err
					}

					dialer, err := proxy.NewDialer(_env.IsLocalDevelopmentMode())
					if err != nil {
						return err
					}

					a := actuator.NewService(_env.Logger(), dialer, buckets, clusters, manifests, m)

					sigterm := make(chan os.Signal, 1)
					done := make(chan struct{})
					signal.Notify(sigterm, syscall.SIGTERM)

					go a.Run(ctx.Context, stop, done)

					<-sigterm
					log.Print("received SIGTERM")
					close(stop)
					//<-done

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
