package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
	"github.com/jim-minter/rp/pkg/env"
)

func run(ctx context.Context) error {
	for _, key := range []string{
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s resourceid", os.Args[0])
	}

	databaseAccount, masterKey, err := env.CosmosDB(ctx)
	if err != nil {
		return err
	}

	dbc, err := cosmosdb.NewDatabaseClient(http.DefaultClient, databaseAccount, masterKey)
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
	if err := run(context.Background()); err != nil {
		panic(err)
	}
}
