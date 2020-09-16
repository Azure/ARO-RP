package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var GitCommit = "unknown"

// InstallStream describes stream we are defaulting to for all new clusters
var InstallStream = Stream{
	Version:    NewVersion(4, 4, 20),
	PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:d14e6d01c7a41f7d76c42e100207a3f48bb416fd1a863dad4e2708b6c5a9f366",
	MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:a7fe9502442180833dc0805f84fe5177c510d16fcc6d5a77767a0bb435b65e19",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []Stream{
		InstallStream,
		{
			Version:    NewVersion(4, 3, 35),
			PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:557107b1a5911d73d9fa749cf103cc3b45d688bf0da991471279042eec84f830",
			MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:58bba19084b2c4ead3bad2575491cd98c2475c80934c75cc01da2b91ce3fc75e",
		},
	}
)

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acr string) string {
	return acr + ".azurecr.io/fluentbit:1.3.9-1"
}

// MdmImage contains the location of the MDM container image
func MdmImage(acr string) string {
	return acr + ".azurecr.io/genevamdm:master_41"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acr string) string {
	return acr + ".azurecr.io/genevamdsd:master_295"
}
