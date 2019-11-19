package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	_ "github.com/jim-minter/rp/pkg/api/v20191231preview"
	"github.com/jim-minter/rp/pkg/backend"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
	"github.com/jim-minter/rp/pkg/frontend"
	uuid "github.com/satori/go.uuid"
)

var (
	gitCommit = "unknown"
)

func run(ctx context.Context, log *logrus.Entry) error {
	uuid := uuid.NewV4()
	log.Printf("starting, git commit %s, uuid %s", gitCommit, uuid)

	for _, key := range []string{
		"LOCATION",
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	env, err := env.NewEnv(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"))
	if err != nil {
		return err
	}

	db, err := database.NewOpenShiftClusters(ctx, env, uuid, "OpenShiftClusters", "OpenShiftClusterDocuments")
	if err != nil {
		return err
	}

	sigterm := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	b, err := backend.NewBackend(ctx, log.WithField("component", "backend"), env, db)
	if err != nil {
		return err
	}

	f, err := frontend.NewFrontend(ctx, log.WithField("component", "frontend"), env, db)
	if err != nil {
		return err
	}

	log.Print("listening")

	go b.Run(stop)
	go f.Run(stop)

	<-sigterm
	log.Print("received SIGTERM")
	close(stop)

	select {}
}

func main() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	log := logrus.NewEntry(logrus.StandardLogger())

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
