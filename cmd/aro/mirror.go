package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
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
	env, err := env.NewCoreForCI(ctx, log)
	if err != nil {
		return err
	}

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

	acrDomainSuffix := "." + env.Environment().ContainerRegistryDNSSuffix

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

	var releases []pkgmirror.Node
	if len(flag.Args()) == 1 {
		log.Print("reading release graph")
		releases, err = pkgmirror.AddFromGraph(version.NewVersion(4, 3))
		if err != nil {
			return err
		}
	} else {
		for _, arg := range flag.Args()[1:] {
			releases = append(releases, pkgmirror.Node{
				Version: arg,
				Payload: arg,
			})
		}
	}

	var errorOccurred bool
	for _, release := range releases {
		log.Printf("mirroring release %s", release.Version)
		err = pkgmirror.Mirror(ctx, log, dstAcr+acrDomainSuffix, release.Payload, dstAuth, srcAuthQuay)
		if err != nil {
			log.Errorf("%s: %s\n", release, err)
			errorOccurred = true
		}
	}

	if len(flag.Args()) == 1 {
		for _, ref := range []string{
			version.MdsdImage("linuxgeneva-microsoft" + acrDomainSuffix),
			version.MdmImage("linuxgeneva-microsoft" + acrDomainSuffix),
		} {
			log.Printf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref))
			err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref), ref, dstAuth, srcAuthGeneva)
			if err != nil {
				log.Errorf("%s: %s\n", ref, err)
				errorOccurred = true
			}
		}

		for _, ref := range []string{
			"registry.redhat.io/rhel7/support-tools:latest",
			"registry.redhat.io/rhel8/support-tools:latest",
		} {
			log.Printf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref))
			err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref), ref, dstAuth, srcAuthRedhat)
			if err != nil {
				log.Errorf("%s: %s\n", ref, err)
				errorOccurred = true
			}
		}
	}

	log.Print("done")

	if errorOccurred {
		return fmt.Errorf("an error occurred")
	}

	return nil
}
