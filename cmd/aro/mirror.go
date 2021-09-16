package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	pkgmirror "github.com/Azure/ARO-RP/pkg/mirror"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// These are versions that need to be skipped because they are unable
// to be mirrored
var doNotMirrorTags = map[string]struct{}{
	"4.8.8":  {}, // release points to unreachable link
	"4.7.27": {}, // release points to unreachable link
}

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
		"SRC_AUTH_QUAY",
		"SRC_AUTH_REDHAT",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	env, err := env.NewCoreForCI(ctx, log)
	if err != nil {
		return err
	}

	acrDomainSuffix := "." + env.Environment().ContainerRegistryDNSSuffix

	dstAuth, err := getAuth("DST_AUTH")
	if err != nil {
		return err
	}

	dstAcr := os.Getenv("DST_ACR_NAME")
	srcAcrGenevaOverride := os.Getenv("SRC_ACR_NAME_GENEVA_OVERRIDE") // Optional

	srcAuthRedhat, err := getAuth("SRC_AUTH_REDHAT")
	if err != nil {
		return err
	}

	var srcAuthGeneva *types.DockerAuthConfig
	if os.Getenv("SRC_AUTH_GENEVA") != "" {
		srcAuthGeneva, err = getAuth("SRC_AUTH_GENEVA") // Optional.  Needed for situations where ACR doesn't allow anonymous pulls
		if err != nil {
			return err
		}
	}

	var errorOccurred bool
	srcAcrGeneva := "linuxgeneva-microsoft" + acrDomainSuffix

	if srcAcrGenevaOverride != "" {
		srcAcrGeneva = srcAcrGenevaOverride
	}

	mirrorImages := []string{
		version.MdsdImage(srcAcrGeneva),
		version.MdmImage(srcAcrGeneva),
	}

	for _, ref := range mirrorImages {
		log.Printf("mirroring %s -> %s", ref, pkgmirror.DestLastIndex(dstAcr+acrDomainSuffix, ref))
		err = pkgmirror.Copy(ctx, pkgmirror.DestLastIndex(dstAcr+acrDomainSuffix, ref), ref, dstAuth, srcAuthGeneva)
		if err != nil {
			log.Errorf("%s: %s\n", ref, err)
			errorOccurred = true
		}
	}

	for _, ref := range []string{
		"registry.redhat.io/rhel7/support-tools:latest",
		"registry.redhat.io/rhel8/support-tools:latest",
		"registry.redhat.io/openshift4/ose-tools-rhel7:latest",
		"registry.redhat.io/openshift4/ose-tools-rhel8:latest",
	} {
		log.Printf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref))
		err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref), ref, dstAuth, srcAuthRedhat)
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
