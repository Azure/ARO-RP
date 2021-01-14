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
	Version:  NewVersion(4, 6, 12),
	PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:5c3618ab914eb66267b7c552a9b51c3018c3a8f8acf08ce1ff7ae4bfdd3a82bd",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []*Stream{
		InstallStream,
		{
			Version:  NewVersion(4, 5, 27),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:412276155bfe186c35322a788321ebf110130a272e18f55a1a2510f15ee0bb04",
		},
		{
			Version:  NewVersion(4, 4, 31),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:52ca6e018793f068f994ff1e85d86283fd4a9875390dffde55c97fd59d03a009",
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
	return acrDomain + "/genevamdm:master_52"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acrDomain string) string {
	return acrDomain + "/genevamdsd:master_20201121.1"
}

func RouteFixImage(acrDomain string) string {
	return acrDomain + "/routefix:c5c4a5db"
}
