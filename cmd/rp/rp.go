package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jim-minter/rp/pkg/queue/leaser"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	_ "github.com/jim-minter/rp/pkg/api/v20191231preview"
	"github.com/jim-minter/rp/pkg/backend"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
	"github.com/jim-minter/rp/pkg/frontend"
	"github.com/jim-minter/rp/pkg/queue"
	"github.com/jim-minter/rp/pkg/queue/forwarder"
)

func run(log *logrus.Entry) error {
	for _, key := range []string{
		"COSMOSDB_ACCOUNT",
		"COSMOSDB_KEY",
		"DOMAIN",
		"DOMAIN_RESOURCEGROUP",
		"STORAGE_ACCOUNT",
		"STORAGE_KEY",
		"LOCATION",
		"HOME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	dbc, err := cosmosdb.NewDatabaseClient(http.DefaultClient, os.Getenv("COSMOSDB_ACCOUNT"), os.Getenv("COSMOSDB_KEY"))
	if err != nil {
		return err
	}
	db := database.NewOpenShiftClusters(dbc, "OpenShiftClusters", "OpenShiftClusterDocuments")

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	sigterm := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	{
		log := log.WithField("component", "backend")
		q, err := queue.NewQueue(log, os.Getenv("STORAGE_ACCOUNT"), os.Getenv("STORAGE_KEY"), "openshiftclusterdocuments")
		if err != nil {
			return err
		}
		go backend.NewBackend(log, authorizer, q, db).Run(done)
	}

	{
		log := log.WithField("component", "queue")
		q, err := queue.NewQueue(log, os.Getenv("STORAGE_ACCOUNT"), os.Getenv("STORAGE_KEY"), "openshiftclusterdocuments")
		if err != nil {
			return err
		}
		l := leaser.NewLeaser(log, dbc, "OpenShiftClusters", "Leases", "forwarder", 10*time.Second, 60*time.Second)
		go forwarder.NewForwarder(log, q, db, l).Run(done)
	}

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	log.Print("listening")

	go frontend.NewFrontend(log.WithField("component", "frontend"), l, db, api.APIs).Run(done)

	<-sigterm
	log.Print("received SIGTERM")
	close(done)

	select {}
}

func main() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	log := logrus.NewEntry(logrus.StandardLogger())

	if err := run(log); err != nil {
		log.Fatal(err)
	}
}
