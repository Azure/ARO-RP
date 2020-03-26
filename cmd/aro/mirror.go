package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

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
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	dstauth, err := getAuth("DST_AUTH")
	if err != nil {
		return err
	}

	dstAcr, _ := os.LookupEnv("DST_ACR_NAME")

	srcauthGeneva, err := getAuth("SRC_AUTH_GENEVA")
	if err != nil {
		return err
	}

	srcauthQuay, err := getAuth("SRC_AUTH_QUAY")
	if err != nil {
		return err
	}

	log.Print("reading Cincinnati graph")
	releases, err := pkgmirror.AddFromGraph("stable", pkgmirror.Version{4, 3})
	if err != nil {
		return err
	}

	// ensure we mirror the version at which we are creating clusters, even if
	// it isn't in the Cincinnati graph yet
	var found bool
	for _, release := range releases {
		if release.Version == version.OpenShiftVersion {
			found = true
			break
		}
	}

	if !found {
		releases = append(releases, pkgmirror.Node{
			Version: version.OpenShiftVersion,
			Payload: strings.Replace(version.OpenShiftPullSpec(dstAcr), dstAcr+".azurecr.io/", "quay.io/", 1),
		})
	}

	var errorOccurred bool
	for _, release := range releases {
		log.Printf("mirroring release %s", release.Version)
		err = pkgmirror.Mirror(ctx, log, dstAcr+".azurecr.io", release.Payload, dstauth, srcauthQuay)
		if err != nil {
			log.Errorf("%s: %s\n", release, err)
			errorOccurred = true
		}
	}

	for _, ref := range []string{
		"linuxgeneva-microsoft.azurecr.io/genevamdsd:master_266",
		"linuxgeneva-microsoft.azurecr.io/genevamdm:master_35",
	} {
		log.Printf("mirroring %s", ref)
		err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+".azurecr.io", ref), ref, dstauth, srcauthGeneva)
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
