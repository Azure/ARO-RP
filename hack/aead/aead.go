package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func run(ctx context.Context, log *logrus.Entry) error {
	fileName := flag.String("file", "-", "File to read. '-' for stdin.")

	flag.Parse()

	var (
		file io.Reader
		err  error
		v    string
	)

	if *fileName == "-" {
		file = os.Stdin
	} else {
		file, err = os.Open(*fileName)
		if err != nil {
			return err
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		v = v + scanner.Text() + "\n"
	}

	_env, err := env.NewCore(ctx, log)
	if err != nil {
		return err
	}

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return err
	}

	serviceKeyvaultURI, err := keyvault.URI(_env, env.ServiceKeyvaultSuffix)
	if err != nil {
		return err
	}

	serviceKeyvault := keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	aead, err := encryption.NewMulti(ctx, serviceKeyvault, env.EncryptionSecretV2Name, env.EncryptionSecretName)
	if err != nil {
		return err
	}

	b, err := aead.Seal([]byte(api.SecureString(v)))
	if err != nil {
		return err
	}

	fmt.Println(base64.StdEncoding.EncodeToString(b))
	return nil
}

func main() {
	log := utillog.GetLogger()

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
