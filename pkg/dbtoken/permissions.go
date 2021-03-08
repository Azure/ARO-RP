package dbtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"os"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func ConfigurePermissions(ctx context.Context, dbid string, userc cosmosdb.UserClient) error {
	gateway := os.Getenv("AZURE_GATEWAY_SERVICE_PRINCIPAL_ID")

	_, err := userc.Create(ctx, &cosmosdb.User{
		ID: gateway,
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		return err
	}

	permc := cosmosdb.NewPermissionClient(userc, gateway)
	_, err = permc.Create(ctx, &cosmosdb.Permission{
		ID:             "gateway",
		PermissionMode: cosmosdb.PermissionModeRead,
		Resource:       "dbs/" + dbid + "/colls/Gateway",
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusConflict) {
		return err
	}

	return nil
}
