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

	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	pkgmirror "github.com/Azure/ARO-RP/pkg/mirror"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// These are versions that need to be skipped because they are unable
// to be mirrored
var doNotMirrorTags = map[string]struct{}{
	"4.8.8": {}, // release points to unreachable link
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
	err := env.ValidateVars(
		"DST_AUTH",
		"DST_ACR_NAME",
		"SRC_AUTH_QUAY",
		"SRC_AUTH_REDHAT")

	if err != nil {
		return err
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

	// We can lose visibility of early image mirroring errors because logs are trimmed in the output of Ev2 pipelines.
	// If images fail to mirror, those errors need to be returned together and logged at the end of the execution.
	var imageMirroringErrors []string

	for _, ref := range []string{

		// https://mcr.microsoft.com/en-us/product/azure-cli/about
		"mcr.microsoft.com/azure-cli:cbl-mariner2.0",
		"mcr.microsoft.com/azure-cli:azurelinux3.0",

		// https://catalog.redhat.com/software/containers/rhel8/support-tools/5ba3eaf9bed8bd6ee819b78b
		// https://catalog.redhat.com/software/containers/rhel9/support-tools/615be213075b022acc111bf9
		"registry.redhat.io/rhel8/support-tools:latest",
		"registry.redhat.io/rhel9/support-tools:latest",

		// https://catalog.redhat.com/software/containers/openshift4/ose-tools-rhel8/5f748d3399cc5b9e7c1a8747
		"registry.redhat.io/openshift4/ose-tools-rhel8:v4.12",
		"registry.redhat.io/openshift4/ose-tools-rhel8:v4.13",
		"registry.redhat.io/openshift4/ose-tools-rhel8:v4.14",
		"registry.redhat.io/openshift4/ose-tools-rhel8:v4.15",

		// https://catalog.redhat.com/software/containers/openshift4/ose-cli-rhel9/6528096620ebdcf82af4cbf9
		"registry.redhat.io/openshift4/ose-cli-rhel9:v4.16",
		"registry.redhat.io/openshift4/ose-cli-rhel9:v4.17",
		"registry.redhat.io/openshift4/ose-cli-rhel9:latest",

		// https://catalog.redhat.com/software/containers/ubi9/ubi-minimal/615bd9b4075b022acc111bf5
		"registry.access.redhat.com/ubi9/ubi-minimal:latest",

		// https://catalog.redhat.com/software/containers/ubi8/nodejs-18/6278e5c078709f5277f26998
		"registry.access.redhat.com/ubi8/nodejs-18:latest",
		// https://catalog.redhat.com/software/containers/ubi9/nodejs-18/62e8e7ed22d1d3c2dfe2ca01
		"registry.access.redhat.com/ubi9/nodejs-18:latest",

		// https://quay.io/repository/app-sre/managed-upgrade-operator?tab=tags
		// https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/osd-operators/cicd/saas/saas-managed-upgrade-operator.yaml?ref_type=heads
		"quay.io/app-sre/managed-upgrade-operator:v0.1.1202-g118c178",

		// https://quay.io/repository/app-sre/hive?tab=tags
		"quay.io/app-sre/hive:87bff5947f",
	} {
		log.Printf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref))

		srcAuth := srcAuthRedhat
		if strings.Index(ref, "quay.io") == 0 {
			srcAuth = srcAuthQuay
		}

		err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr+acrDomainSuffix, ref), ref, dstAuth, srcAuth)
		if err != nil {
			imageMirroringErrors = append(imageMirroringErrors, fmt.Sprintf("%s: %s\n", ref, err))
		}
	}

	// OCP release mirroring
	var releases []pkgmirror.Node
	if len(flag.Args()) == 1 {
		log.Print("reading release graph")
		releases, err = pkgmirror.AddFromGraph(version.NewVersion(4, 14))
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

	for _, release := range releases {
		if _, ok := doNotMirrorTags[release.Version]; ok {
			log.Printf("skipping mirror of release %s", release.Version)
			continue
		}
		log.Printf("mirroring release %s", release.Version)
		err = pkgmirror.Mirror(ctx, log, dstAcr+acrDomainSuffix, release.Payload, dstAuth, srcAuthQuay)
		if err != nil {
			imageMirroringErrors = append(imageMirroringErrors, fmt.Sprintf("%s: %s\n", release, err))
		}
	}

	log.Print("done")

	if imageMirroringErrors != nil {
		return fmt.Errorf("failed to mirror image/s\n%s", strings.Join(imageMirroringErrors, "\n"))
	}

	return nil
}
