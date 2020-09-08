package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/deploy/generator"
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

	_env, err := env.NewEnvLite(ctx, log)
	if err != nil {
		return err
	}

	rpKVAuthorizer, err := env.RPAuthorizer(azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	serviceKeyvaultURI, err := env.GetVaultURI(ctx, _env, generator.ServiceKeyVaultTagValue)
	if err != nil {
		return err
	}

	kv := keyvault.NewManager(rpKVAuthorizer, serviceKeyvaultURI)

	bundle, err := kv.GetSecret(ctx, env.EncryptionSecretName, "")
	if err != nil {
		return err
	}

	dbKey, err := base64.StdEncoding.DecodeString(*bundle.Value)
	if err != nil {
		return err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, dbKey)
	if err != nil {
		return err
	}

	dbc, err := database.NewDatabaseClient(ctx, log.WithField("component", "database"), _env, &noop.Noop{}, cipher)
	if err != nil {
		return err
	}

	openShiftClusters, err := database.NewOpenShiftClusters(ctx, _env, dbc, "")
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
