package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func run(ctx context.Context, log *logrus.Entry) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s resourceid", os.Args[0])
	}

	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	rpKVAuthorizer, err := _env.NewRPAuthorizer(_env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	serviceKeyvaultURI, err := keyvault.URI(_env, env.ServiceKeyvaultSuffix)
	if err != nil {
		return err
	}

	serviceKeyvault := keyvault.NewManager(rpKVAuthorizer, serviceKeyvaultURI)

	key, err := serviceKeyvault.GetBase64Secret(ctx, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	aead, err := encryption.NewXChaCha20Poly1305(ctx, key)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(ctx, log.WithField("component", "database"), _env, &noop.Noop{}, aead)
	if err != nil {
		return err
	}

	openShiftClusters, err := database.NewOpenShiftClusters(ctx, _env.IsLocalDevelopmentMode(), dbc)
	if err != nil {
		return err
	}

	doc, err := openShiftClusters.Get(ctx, strings.ToLower(os.Args[1]))
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(doc)
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
