package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var GitCommit = "unknown"

// InstallStream describes stream we are defaulting to for all new clusters
var InstallStream = Stream{
	Version:    NewVersion(4, 4, 17),
	PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:624e84b5d22cb865ee1be32aa6e3feea99917c6081f7a9c5b1185fc9934d23f3",
	MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:2acc62c38f40bcebc003a6ce8a30ee58f5c1ed6dc0d8514811cc70528d93a65d",
}

// Streams describes list of streams we support for upgrades
var (
	Streams = []Stream{
		InstallStream,
		{
			Version:    NewVersion(4, 3, 31),
			PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:6395ddd44276c4a1d760c77f9f5d8dabf302df7b84afd7b3147c97bdf268ab0f",
			MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:bdb0536aaa8d581990f4e73b6c55d4938536ca697b5370d4627adaf666e6cb66",
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
