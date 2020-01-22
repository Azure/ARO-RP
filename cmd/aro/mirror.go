package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/containers/image/types"
	"github.com/sirupsen/logrus"

	pkgmirror "github.com/Azure/ARO-RP/pkg/mirror"
)

func getAuth(key string) (*types.DockerAuthConfig, error) {
	b, err := base64.StdEncoding.DecodeString(os.Getenv(key))
	if err != nil {
		return nil, err
	}

	return &types.DockerAuthConfig{
		Username: string(b[:bytes.IndexByte(b, ':')]),
		Password: string(b[bytes.IndexByte(b, ':')+1:]),
	}, nil
}

func mirror(ctx context.Context, log *logrus.Entry) error {
	for _, key := range []string{
		"DST_AUTH",
		"SRC_AUTH",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	dstauth, err := getAuth("DST_AUTH")
	if err != nil {
		return err
	}

	srcauth, err := getAuth("SRC_AUTH")
	if err != nil {
		return err
	}

	log.Print("reading Cincinnati graph")
	releases, err := pkgmirror.AddFromGraph("stable", pkgmirror.Version{4, 3})
	if err != nil {
		return err
	}

	releases = append(releases,
		// quay.io/openshift-release-dev/ocp-release:4.3.0-x86_64
		"quay.io/openshift-release-dev/ocp-release@sha256:3a516480dfd68e0f87f702b4d7bdd6f6a0acfdac5cd2e9767b838ceede34d70d",
	)

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
