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
	Version:  NewVersion(4, 6, 17),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:a7b23f38d1e5be975a6b516739689673011bdfa59a7158dc6ca36cefae169c18",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []*Stream{
		InstallStream,
		{
			Version:  NewVersion(4, 5, 31),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:27951dd757d472bf913daaffa548b865e87968831ca6f42c1f6946f7dcf0564e",
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
	return acrDomain + "/fluentbit:1.6.10-1"
}

// MdmImage contains the location of the MDM container image
func MdmImage(acrDomain string) string {
	return acrDomain + "/genevamdm:master_20210201.2"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acrDomain string) string {
	return acrDomain + "/genevamdsd:master_20210201.2"
}
