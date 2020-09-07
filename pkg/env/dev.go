package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

var _ Interface = &dev{}

type dev struct {
	*prod
}

func newDev(ctx context.Context, log *logrus.Entry, instancemetadata instancemetadata.InstanceMetadata) (*dev, error) {
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

	d := &dev{}

	var err error
	d.prod, err = newProd(ctx, log, instancemetadata)
	if err != nil {
		return nil, err
	}

	d.prod.envType = Dev

	return d, nil
}
