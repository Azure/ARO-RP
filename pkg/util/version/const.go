package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var GitCommit = "unknown"

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
	return acr + ".azurecr.io/genevamdm:master_48"
}

// MdsdImage contains the location of the MDSD container image
func MdsdImage(acr string) string {
	return acr + ".azurecr.io/genevamdsd:master_309"
}
