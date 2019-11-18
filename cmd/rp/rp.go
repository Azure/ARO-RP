package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	_ "github.com/jim-minter/rp/pkg/api/v20191231preview"
	"github.com/jim-minter/rp/pkg/backend"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
	"github.com/jim-minter/rp/pkg/frontend"
	uuid "github.com/satori/go.uuid"
)

func run(log *logrus.Entry) error {
	for _, key := range []string{
		"COSMOSDB_ACCOUNT",
		"COSMOSDB_KEY",
		"DOMAIN",
		"LOCATION",
		"RP_RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	uuid := uuid.NewV4()
	log.Printf("starting, uuid %s", uuid)

	dbc, err := cosmosdb.NewDatabaseClient(http.DefaultClient, os.Getenv("COSMOSDB_ACCOUNT"), os.Getenv("COSMOSDB_KEY"))
	if err != nil {
		return err
	}

	db, err := database.NewOpenShiftClusters(uuid, dbc, "OpenShiftClusters", "OpenShiftClusterDocuments")
	if err != nil {
		return err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	sigterm := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(sigterm, syscall.SIGTERM)

	go backend.NewBackend(log.WithField("component", "backend"), authorizer, db).Run(stop)

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	log.Print("listening")

	go frontend.NewFrontend(log.WithField("component", "frontend"), l, db, api.APIs).Run(stop)

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

	if err := run(log); err != nil {
		log.Fatal(err)
	}
}
