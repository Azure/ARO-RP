package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

type Type int

const (
	Prod Type = iota
	Dev
	Int
)

const (
	RPFirstPartySecretName       = "rp-firstparty"
	RPServerSecretName           = "rp-server"
	ClusterLoggingSecretName     = "cluster-mdsd"
	EncryptionSecretName         = "encryption-key"
	FrontendEncryptionSecretName = "fe-encryption-key"
	RPLoggingSecretName          = "rp-mdsd"
	RPMonitoringSecretName       = "rp-mdm"
)

type Interface interface {
	Lite

	Domain() string
	ManagedDomain(string) (string, error)
	Zones(vmSize string) ([]string, error)
	ACRResourceID() string
	ACRName() string
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	switch envType() {
	case Dev:
		log.Warn("running in development mode")

		for _, key := range []string{
			"AZURE_RP_CLIENT_ID",
			"AZURE_RP_CLIENT_SECRET",
			"AZURE_FP_CLIENT_ID",
			"AZURE_TENANT_ID",
		} {
			if _, found := os.LookupEnv(key); !found {
				return nil, fmt.Errorf("environment variable %q unset", key)
			}
		}

	case Int:
		log.Warn("running in int mode")
	}

	im, err := newInstanceMetadata(ctx)
	if err != nil {
		return nil, err
	}

	return newProd(ctx, log, im, envType())
}

func newInstanceMetadata(ctx context.Context) (instancemetadata.InstanceMetadata, error) {
	if envType() == Dev {
		return instancemetadata.NewDev()
	}

	return instancemetadata.NewProd(ctx)
}

func envType() Type {
	switch {
	case strings.ToLower(os.Getenv("RP_MODE")) == "development":
		return Dev
	case strings.ToLower(os.Getenv("RP_MODE")) == "int":
		return Int
	default:
		return Prod
	}
}
