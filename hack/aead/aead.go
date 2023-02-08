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

const (
	KeyVaultPrefix = "KEYVAULT_PREFIX"
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

	msiKVAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().KeyVaultScope)
	if err != nil {
		return err
	}

	if err := ValidateVars(KeyVaultPrefix); err != nil {
		return err
	}
	keyVaultPrefix := os.Getenv(KeyVaultPrefix)
	serviceKeyvaultURI := keyvault.URI(_env, env.ServiceKeyvaultSuffix, keyVaultPrefix)
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

// ValidateVars iterates over all the elements of vars and
// if it does not exist an environment variable with that name, it will return an error.
// Otherwise it returns nil.
func ValidateVars(vars ...string) error {
	for _, v := range vars {
		if _, found := os.LookupEnv(v); !found {
			return fmt.Errorf("environment variable %q unset", v)
		}
	}
	return nil
}
