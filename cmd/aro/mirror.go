package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"flag"
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

var (
	openShiftVersionToMirror string
	mirrorOpenShiftImages    bool
	mirrorMicrosoftImages    bool
	mirrorBaseImages         bool
	mirrorAppSREImages       bool
)

func mirrorFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("mirror", flag.ContinueOnError)

	flags.StringVar(&openShiftVersionToMirror, "openShiftVersion", "", "OpenShift major/minor version to mirror, 4.6+ if not specified")
	flags.BoolVar(&mirrorOpenShiftImages, "mirrorOpenShiftImages", false, "Whether to mirror OpenShift releases, default false")
	flags.BoolVar(&mirrorMicrosoftImages, "mirrorMicrosoftImages", false, "Whether to mirror Microsoft Images (e.g. MDSD), default false")
	flags.BoolVar(&mirrorBaseImages, "mirrorBaseImages", false, "Whether to mirror base images (e.g. toolboxes, go-toolset), default false")
	flags.BoolVar(&mirrorAppSREImages, "mirrorAppSREImages", false, "Whether to mirror AppSRE images (MUO, Hive), default false")

	return flags
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
	mirrorFlags := mirrorFlags()
	err := mirrorFlags.Parse(flag.Args()[1:])
	if err != nil {
		return err
	}

	// If no mirroring has been selected, error out
	if !mirrorOpenShiftImages && !mirrorMicrosoftImages && !mirrorBaseImages && !mirrorAppSREImages {
		return errors.New("select the images to mirror, try --help")
	}

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

	srcAuthQuay, err := getAuth("SRC_AUTH_QUAY")
	if err != nil {
		return err
	}

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

	errorOccurred := false
	m := pkgmirror.New(env, log, dstAcr+acrDomainSuffix, dstAuth)

	if mirrorOpenShiftImages {
		if openShiftVersionToMirror == "" {
			return errors.New("if mirroring OpenShift images, --openShiftVersion is required")
		}
		// Limit the mirroring to the desired minor version, so that we can run them individually
		desiredMinorVersion, err := version.ParseMinorVersion(openShiftVersionToMirror)
		if err != nil {
			return err
		}

		err = m.MirrorOpenShiftVersion(ctx, srcAuthQuay, desiredMinorVersion, doNotMirrorTags)
		if err == pkgmirror.ErrMirror {
			errorOccurred = true
		} else {
			return err
		}
	}

	if mirrorMicrosoftImages {
		srcAcrGeneva := "linuxgeneva-microsoft" + acrDomainSuffix
		if srcAcrGenevaOverride != "" {
			srcAcrGeneva = srcAcrGenevaOverride
		}

		refs := []string{
			version.MdsdImage(srcAcrGeneva),
			version.MdmImage(srcAcrGeneva),
		}

		err := m.MirrorImageRefs(ctx, srcAuthGeneva, refs)
		if err == pkgmirror.ErrMirror {
			errorOccurred = true
		} else {
			return err
		}
	}

	if mirrorBaseImages {
		refs := []string{
			"registry.redhat.io/rhel7/support-tools:latest",
			"registry.redhat.io/rhel8/support-tools:latest",
			"registry.redhat.io/openshift4/ose-tools-rhel7:latest",
			"registry.redhat.io/openshift4/ose-tools-rhel8:latest",
			"registry.access.redhat.com/ubi7/ubi-minimal:latest",
			"registry.access.redhat.com/ubi8/ubi-minimal:latest",
			"registry.access.redhat.com/ubi8/nodejs-14:latest",
			"registry.access.redhat.com/ubi7/go-toolset:1.16.12",
			"registry.access.redhat.com/ubi8/go-toolset:1.17.7",
			"mcr.microsoft.com/azure-cli:latest",
		}

		err := m.MirrorImageRefs(ctx, srcAuthRedhat, refs)
		if err == pkgmirror.ErrMirror {
			errorOccurred = true
		} else {
			return err
		}
	}

	if mirrorAppSREImages {
		refs := []string{
			"quay.io/app-sre/managed-upgrade-operator:v0.1.856-eebbe07",
			"quay.io/app-sre/hive:fec14dc",
		}

		err := m.MirrorImageRefs(ctx, srcAuthQuay, refs)
		if err == pkgmirror.ErrMirror {
			errorOccurred = true
		} else {
			return err
		}
	}

	log.Print("done")

	if errorOccurred {
		return fmt.Errorf("an error occurred")
	}

	return nil
}
