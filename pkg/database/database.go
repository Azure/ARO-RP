package database

import (
	"context"
	"net/http"

	"github.com/jim-minter/rp/pkg/database/cosmosdb"

	"github.com/jim-minter/rp/pkg/env"
	uuid "github.com/satori/go.uuid"
)

// Database represents a database
type Database struct {
	OpenShiftClusters OpenShiftClusters
}

// NewDatabase returns a new Database
func NewDatabase(ctx context.Context, env env.Interface, uuid uuid.UUID, dbid string) (db *Database, err error) {
	databaseAccount, masterKey, err := env.CosmosDB(ctx)
	if err != nil {
		return nil, err
	}

	dbc, err := cosmosdb.NewDatabaseClient(http.DefaultClient, databaseAccount, masterKey)
	if err != nil {
		return nil, err
	}

	db = &Database{}
	db.OpenShiftClusters, err = NewOpenShiftClusters(ctx, uuid, dbc, dbid, "OpenShiftClusterDocuments")
	if err != nil {
		return nil, err
	}
	return db, nil
}
