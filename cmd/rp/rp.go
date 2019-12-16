package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/backend"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
	"github.com/jim-minter/rp/pkg/frontend"
	utillog "github.com/jim-minter/rp/pkg/util/log"
)

var (
	gitCommit = "unknown"
)

func run(ctx context.Context, log *logrus.Entry) error {
	uuid := uuid.NewV4()
	log.Printf("starting, git commit %s, uuid %s", gitCommit, uuid)

	env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(ctx, env, uuid, "ARO")
	if err != nil {
		return err
	}

	sigterm := make(chan os.Signal, 1)
	stop := make(chan struct{})
	done := make(chan struct{})
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
	go f.Run(stop, done)

	<-sigterm
	log.Print("received SIGTERM")
	close(stop)
	<-done

	return nil
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
