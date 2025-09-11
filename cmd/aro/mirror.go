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
	"time"

	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/Azure/ARO-RP/pkg/env"
	pkgmirror "github.com/Azure/ARO-RP/pkg/mirror"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcontainerregistry"
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

func mirror(ctx context.Context, _log *logrus.Entry) error {
	err := env.ValidateVars(
		"DST_ACR_NAME",
		"SRC_AUTH_QUAY",
		"SRC_AUTH_REDHAT")

	if err != nil {
		return err
	}

	var _env env.Core
	var tokenCredential azcore.TokenCredential
	if os.Getenv("AZURE_EV2") != "" {
		var err error
		_env, err = env.NewCore(ctx, _log, env.SERVICE_MIRROR)
		if err != nil {
			return err
		}
		options := _env.Environment().ManagedIdentityCredentialOptions()
		// use specific user-assigned managed identity if set
		if os.Getenv("AZURE_CLIENT_ID") != "" {
			options.ID = azidentity.ClientID(os.Getenv("AZURE_CLIENT_ID"))
		}
		tokenCredential, err = azidentity.NewManagedIdentityCredential(options)
		if err != nil {
			return err
		}
	} else {
		err := env.ValidateVars(
			"AZURE_CLIENT_ID",
			"AZURE_CLIENT_SECRET",
			"AZURE_SUBSCRIPTION_ID",
			"AZURE_TENANT_ID")

		if err != nil {
			return err
		}

		_env, err = env.NewCoreForCI(ctx, _log, env.SERVICE_MIRROR)
		if err != nil {
			return err
		}
		options := _env.Environment().EnvironmentCredentialOptions()
		tokenCredential, err = azidentity.NewEnvironmentCredential(options)
		if err != nil {
			return err
		}
	}
	env := _env

	acrDomainSuffix := "." + env.Environment().ContainerRegistryDNSSuffix

	dstAcr := os.Getenv("DST_ACR_NAME") + acrDomainSuffix
	acrAuthenticationClient, err := azcontainerregistry.NewAuthenticationClient(fmt.Sprintf("https://%s", dstAcr), env.Environment().AzureClientOptions())
	if err != nil {
		return err
	}

	acrauth := pkgmirror.NewAcrAuth(dstAcr, env, tokenCredential, acrAuthenticationClient)

	srcAuthQuay, err := getAuth("SRC_AUTH_QUAY")
	if err != nil {
		return err
	}

	srcAuthRedhat, err := getAuth("SRC_AUTH_REDHAT")
	if err != nil {
		return err
	}

	mirrorLog := _env.LoggerForComponent("mirror")

	// We can lose visibility of early image mirroring errors because logs are trimmed in the output of Ev2 pipelines.
	// If images fail to mirror, those errors need to be returned together and logged at the end of the execution.
	var imageMirroringSummary []string

	for _, ref := range []string{

		// https://hub.docker.com/_/fedora
		"registry.fedoraproject.org/fedora:42",

		// https://hub.docker.com/r/selenium/standalone-edge
		"docker.io/selenium/standalone-edge:4.10.0",

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

		// https://catalog.redhat.com/software/containers/ubi9/ubi-micro/615bd9b4075b022acc111bf5
		"registry.access.redhat.com/ubi9/ubi-micro:latest",

		// https://catalog.redhat.com/software/containers/ubi9/toolbox/615bd9b4075b022acc111bf5
		"registry.access.redhat.com/ubi9/toolbox:latest",

		// https://catalog.redhat.com/software/containers/ubi8/nodejs-18/6278e5c078709f5277f26998
		"registry.access.redhat.com/ubi8/nodejs-18:latest",
		// https://catalog.redhat.com/software/containers/ubi9/nodejs-18/62e8e7ed22d1d3c2dfe2ca01
		"registry.access.redhat.com/ubi9/nodejs-18:latest",

		// https://quay.io/repository/app-sre/managed-upgrade-operator?tab=tags
		// https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/osd-operators/cicd/saas/saas-managed-upgrade-operator.yaml?ref_type=heads
		"quay.io/app-sre/managed-upgrade-operator:v0.1.1202-g118c178",

		// https://quay.io/repository/app-sre/hive?tab=tags
		"quay.io/app-sre/hive:8796c4f534",

		// https://quay.io/repository/openshift/aro-must-gather?tab=tags  
		"quay.io/openshift/aro-must-gather:latest",


		// OpenShift Automated Release Tooling partner images
		// These images are re-tagged versions of the images that OpenShift uses to build internally, mirrored for use in building ARO-RP in CI and ev2
		"quay.io/openshift-release-dev/golang-builder--partner-share:rhel-9-golang-1.23-openshift-4.19",
		"quay.io/openshift-release-dev/golang-builder--partner-share:rhel-9-golang-1.24-openshift-4.20",
	} {
		l := mirrorLog.WithField("payload", ref)
		startTime := time.Now()
		l.Debugf("mirroring %s -> %s", ref, pkgmirror.Dest(dstAcr, ref))

		srcAuth := srcAuthRedhat
		if strings.Index(ref, "quay.io") == 0 {
			srcAuth = srcAuthQuay
		}

		dstAuth, err := acrauth.Get(ctx)
		if err != nil {
			return err
		}
		err = pkgmirror.Copy(ctx, pkgmirror.Dest(dstAcr, ref), ref, dstAuth, srcAuth)
		l.WithError(err).WithField("duration", time.Since(startTime)).Printf("mirroring completed")
		if err != nil {
			imageMirroringSummary = append(imageMirroringSummary, fmt.Sprintf("%s: %s", ref, err))
		}
	}

	// OCP release mirroring
	var releases []pkgmirror.Node
	if len(flag.Args()) == 1 {
		mirrorLog.Print("reading release graph")
		releases, err = pkgmirror.AddFromGraph(version.NewVersion(4, 12))
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
		l := mirrorLog.WithFields(logrus.Fields{"release": release.Version, "payload": release.Payload})
		if _, ok := doNotMirrorTags[release.Version]; ok {
			l.Printf("skipping mirror due to hard-coded deny list")
			continue
		}
		l.Debugf("mirroring release")
		dstAuth, err := acrauth.Get(ctx)
		if err != nil {
			return err
		}
		c, err := pkgmirror.Mirror(ctx, l, dstAcr, release.Payload, dstAuth, srcAuthQuay)
		imageMirroringSummary = append(imageMirroringSummary, fmt.Sprintf("%s (%d)", release.Version, c))
		if err != nil {
			imageMirroringSummary = append(imageMirroringSummary, fmt.Sprintf("Error on %s: %s", release, err))
		}
	}
	fmt.Print("==========\nSummary\n==========\n", strings.Join(imageMirroringSummary, "\n"))
	mirrorLog.Print("done")

	return nil
}
