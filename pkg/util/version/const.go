package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"

	"github.com/Azure/ARO-RP/pkg/api"
)

const InstallArchitectureVersion = api.ArchitectureVersionV2

const (
	DevClusterGenevaLoggingEnvironment   = "Test"
	DevClusterGenevaLoggingConfigVersion = "2.3"
)

var GitCommit = "unknown"

// InstallStream describes stream we are defaulting to for all new clusters
var InstallStream = &Stream{
	Version:  NewVersion(4, 7, 12),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:2029c5779202293f23418d47a1a823c4e4c8539c1ab25e8bda30d48335b4892e",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []*Stream{
		InstallStream,
		{
			Version:  NewVersion(4, 6, 26),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:e867135cd5a09192635b46ccab6ca7543e642378dc72fa22ea54961b05e322f2",
		},
		{
			Version:  NewVersion(4, 5, 38),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:688ca8e365fbd4ee4cc8524e2732395560446d2ff90c3d9b8407b159c571ca1c",
		},
		{
			Version:  NewVersion(4, 4, 33),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:a035dddd8a5e5c99484138951ef4aba021799b77eb9046f683a5466c23717738",
		},
	}
)

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	return acrDomain + "/fluentbit:1.6.10-1"
}

// MdmImage contains the location of the MDM container image
func MdmImage(acrDomain string) string {
	// for the latest version see https://genevamondocs.azurewebsites.net/collect/references/linuxcontainers.html?q=container
	if os.Getenv("GENEVA_MDM_IMAGE_OVERRIDE") != "" {
		return os.Getenv("GENEVA_MDM_IMAGE_OVERRIDE")
	}

	return acrDomain + "/genevamdm:master_20210401.1"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acrDomain string) string {
	// for the latest version see https://genevamondocs.azurewebsites.net/collect/references/linuxcontainers.html?q=container
	if os.Getenv("GENEVA_MDSD_IMAGE_OVERRIDE") != "" {
		return os.Getenv("GENEVA_MDSD_IMAGE_OVERRIDE")
	}

	return acrDomain + "/genevamdsd:master_20210401.1"
}
