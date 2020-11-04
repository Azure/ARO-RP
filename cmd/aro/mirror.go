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
	"github.com/Azure/ARO-RP/pkg/util/version"
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
		"DST_ACR_NAME",
		"SRC_AUTH_GENEVA",
		"SRC_AUTH_QUAY",
		"SRC_AUTH_REDHAT",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	dstAuth, err := getAuth("DST_AUTH")
	if err != nil {
		return err
	}

	dstAcr, _ := os.LookupEnv("DST_ACR_NAME")

	srcAuthGeneva, err := getAuth("SRC_AUTH_GENEVA")
	if err != nil {
		return err
	}

	srcAuthQuay, err := getAuth("SRC_AUTH_QUAY")
	if err != nil {
		return err
	}

	srcAuthRedhat, err := getAuth("SRC_AUTH_REDHAT")
	if err != nil {
		return err
	}

	log.Print("reading release graph")
	releases, err := pkgmirror.AddFromGraph(version.NewVersion(4, 3))
	if err != nil {
		return err
	}

	var errorOccurred bool
	for _, release := range releases {
		log.Printf("mirroring release %s", release.Version)
		err = pkgmirror.Mirror(ctx, log, dstAcr+".azurecr.io", release.Payload, dstAuth, srcAuthQuay)
		if err != nil {
			log.Errorf("%s: %s\n", release, err)
			errorOccurred = true
		}
	}

	for _, ref := range []string{
		version.MdsdImage("linuxgeneva-microsoft"),
		version.MdmImage("linuxgeneva-microsoft"),
	} {
		log.Printf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr+".azurecr.io", ref))
		err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+".azurecr.io", ref), ref, dstAuth, srcAuthGeneva)
		if err != nil {
			log.Errorf("%s: %s\n", ref, err)
			errorOccurred = true
		}
	}

	for _, ref := range []string{
		"registry.redhat.io/rhel7/support-tools:latest",
		"registry.redhat.io/rhel8/support-tools:latest",
	} {
		log.Printf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr+".azurecr.io", ref))
		err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+".azurecr.io", ref), ref, dstAuth, srcAuthRedhat)
		if err != nil {
			log.Errorf("%s: %s\n", ref, err)
			errorOccurred = true
		}
	}

	log.Print("done")

	if errorOccurred {
		return fmt.Errorf("an error occurred")
	}

	return nil
}
