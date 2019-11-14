package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
)

func run() error {
	for _, key := range []string{
		"COSMOSDB_ACCOUNT",
		"COSMOSDB_KEY",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s resourceid", os.Args[0])
	}

	dbc, err := cosmosdb.NewDatabaseClient(http.DefaultClient, os.Getenv("COSMOSDB_ACCOUNT"), os.Getenv("COSMOSDB_KEY"))
	if err != nil {
		return err
	}

	db, err := database.NewOpenShiftClusters(uuid.NewV4(), dbc, "OpenShiftClusters", "OpenShiftClusterDocuments")
	if err != nil {
		return err
	}

	doc, err := db.Get(os.Args[1])
	if err != nil {
		return err
	}

	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
		Indent: 2,
	}
	err = api.AddExtensions(&h.BasicHandle)
	if err != nil {
		return err
	}

	return codec.NewEncoder(os.Stdout, h).Encode(doc)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
