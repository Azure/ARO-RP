package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

const InstallArchitectureVersion = api.ArchitectureVersionV2

var GitCommit = "unknown"

// InstallStream describes stream we are defaulting to for all new clusters
var InstallStream = &Stream{
	Version:  NewVersion(4, 6, 15),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:b70f550e3fa94af2f7d60a3437ec0275194db36f2dc49991da2336fe21e2824c",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []*Stream{
		InstallStream,
		{
			Version:  NewVersion(4, 5, 30),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:48859a33222e306c8077b7d898d07241fd6d6dcf731d0b7cfc9ebf533b2fefa9",
		},
		{
			Version:  NewVersion(4, 4, 33),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:a035dddd8a5e5c99484138951ef4aba021799b77eb9046f683a5466c23717738",
		},
		{
			Version:  NewVersion(4, 3, 40),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:9ff90174a170379e90a9ead6e0d8cf6f439004191f80762764a5ca3dbaab01dc",
		},
	}
)

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	return acrDomain + "/fluentbit:1.3.9-1"
}

// MdmImage contains the location of the MDM container image
func MdmImage(acrDomain string) string {
	return acrDomain + "/genevamdm:master_20210201.2"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acrDomain string) string {
	return acrDomain + "/genevamdsd:master_20210201.2"
}

func RouteFixImage(acrDomain string) string {
	return acrDomain + "/routefix:c5c4a5db"
}
