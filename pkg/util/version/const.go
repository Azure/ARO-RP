package version

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var GitCommit = "unknown"
var ArchitectureVersion = api.ArchitectureVersionV1

// InstallStream describes stream we are defaulting to for all new clusters
var InstallStream = Stream{
	Version:    NewVersion(4, 4, 23),
	PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:0455e0201f475a836f2474d4af7864a55208a33eb6932027f63109bbbd821b65",
	MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:464e897b60cfb39f96a7690d1b8f8972616abe6915f48685b2bbff7d199f8691",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []Stream{
		InstallStream,
		{
			Version:    NewVersion(4, 3, 38),
			PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:59cc585be7b4ad069a18f6f1a3309391e172192744ee65fa6e499c8b337edda4",
			MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:c6c987ea3b52c9b08357e16524f4a023c2e9e07c6c936193ab45ddf34e0fb9ca",
		},
	}
)

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acr string) string {
	return acr + ".azurecr.io/fluentbit:1.3.9-1"
}

// MdmImage contains the location of the MDM container image
func MdmImage(acr string) string {
	return acr + ".azurecr.io/genevamdm:master_48"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acr string) string {
	return acr + ".azurecr.io/genevamdsd:master_309"
}
