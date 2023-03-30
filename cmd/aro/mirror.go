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
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
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

	srcAuthQuay, err := getAuth("SRC_AUTH_QUAY")
	if err != nil {
		return err
	}

	srcAuthRedhat, err := getAuth("SRC_AUTH_REDHAT")
	if err != nil {
		return err
	}

	// Geneva allows anonymous pulls
	var srcAuthGeneva *types.DockerAuthConfig

	var releases []pkgmirror.Node
	if len(flag.Args()) == 1 {
		log.Print("reading release graph")
		releases, err = pkgmirror.AddFromGraph(version.NewVersion(4, 6))
		if err != nil {
			return err
		}
	} else {
		for _, arg := range flag.Args()[1:] {
			if strings.EqualFold(arg, "latest") {
				releases = append(releases, pkgmirror.Node{
					Version: version.DefaultInstallStream.Version.String(),
					Payload: version.DefaultInstallStream.PullSpec,
				})
			} else {
				vers, err := version.ParseVersion(arg)
				if err != nil {
					return err
				}

				node, err := pkgmirror.VersionInfo(vers)
				if err != nil {
					return err
				}

				releases = append(releases, pkgmirror.Node{
					Version: node.Version,
					Payload: node.Payload,
				})
			}
		}
	}

	var errorOccurred bool
	for _, release := range releases {
		if _, ok := doNotMirrorTags[release.Version]; ok {
			log.Printf("skipping mirror of release %s", release.Version)
			continue
		}
		log.Printf("mirroring release %s", release.Version)
		err = pkgmirror.Mirror(ctx, log, dstAcr+acrDomainSuffix, release.Payload, dstAuth, srcAuthQuay)
		if err != nil {
			log.Errorf("%s: %s\n", release, err)
			errorOccurred = true
		}
	}

	// Geneva mirroring from upstream only takes place in Public Cloud, in
	// soverign clouds a separate mirror process mirrors from the public cloud
	if env.Environment().Environment == azure.PublicCloud {
		srcAcrGeneva := "linuxgeneva-microsoft" + acrDomainSuffix

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
	} else {
		log.Printf("skipping Geneva mirroring due to not being in Public")
	}

	for _, ref := range []string{
		"registry.redhat.io/rhel8/support-tools:latest",
		"registry.redhat.io/openshift4/ose-tools-rhel8:latest",
		"registry.access.redhat.com/ubi8/ubi-minimal:latest",
		"registry.access.redhat.com/ubi8/nodejs-14:latest",
		// https://catalog.redhat.com/software/containers/ubi8/go-toolset/5ce8713aac3db925c03774d1
		"registry.access.redhat.com/ubi8/go-toolset:1.17.12",
		"registry.access.redhat.com/ubi8/go-toolset:1.18.4",
		"mcr.microsoft.com/azure-cli:latest",

		// https://quay.io/repository/app-sre/managed-upgrade-operator?tab=tags
		"quay.io/app-sre/managed-upgrade-operator:v0.1.891-3d94c00",
		// https://quay.io/repository/app-sre/hive?tab=tags
		"quay.io/app-sre/hive:fec14dc",
	} {
		log.Printf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref))

		srcAuth := srcAuthRedhat
		if strings.Index(ref, "quay.io") == 0 {
			srcAuth = srcAuthQuay
		}

		err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref), ref, dstAuth, srcAuth)
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
