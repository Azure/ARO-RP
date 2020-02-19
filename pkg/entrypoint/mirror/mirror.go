package mirror

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/containers/image/types"
	"github.com/sirupsen/logrus"

	pkgmirror "github.com/Azure/ARO-RP/pkg/mirror"
)

func start(ctx context.Context, log *logrus.Entry, cfg *Config) error {
	dstauth, err := getAuth(cfg.DstAuth)
	if err != nil {
		return err
	}

	srcauth, err := getAuth(cfg.DstAuth)
	if err != nil {
		return err
	}

	log.Print("reading Cincinnati graph")
	releases, err := pkgmirror.AddFromGraph("stable", pkgmirror.Version{4, 3})
	if err != nil {
		return err
	}

	log.Printf("mirroring %d release(s)", len(releases))

	var errorOccurred bool
	for _, release := range releases {
		err := pkgmirror.Mirror(ctx, log, "arosvc.azurecr.io", release, dstauth, srcauth)
		if err != nil {
			errorOccurred = true
		}
	}

	log.Print("done")

	if errorOccurred {
		return fmt.Errorf("an error occurred")
	}

	return nil
}

func getAuth(value string) (*types.DockerAuthConfig, error) {
	b, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	return &types.DockerAuthConfig{
		Username: string(b[:bytes.IndexByte(b, ':')]),
		Password: string(b[bytes.IndexByte(b, ':')+1:]),
	}, nil

}
