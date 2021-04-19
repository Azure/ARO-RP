package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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
	Version:  NewVersion(4, 6, 21),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:6ae80e777c206b7314732aff542be105db892bf0e114a6757cb9e34662b8f891",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []*Stream{
		InstallStream,
		{
			Version:  NewVersion(4, 5, 36),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:cf535bc369b14350a823490157182e00b658f1b7028e2c80a1be3a6304b20ece",
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
	return acrDomain + "/genevamdm:master_20210401.1"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acrDomain string) string {
	// for the latest version see https://genevamondocs.azurewebsites.net/collect/references/linuxcontainers.html?q=container
	return acrDomain + "/genevamdsd:master_20210401.1"
}
