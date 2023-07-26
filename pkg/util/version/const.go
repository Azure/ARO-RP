package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

const InstallArchitectureVersion = api.ArchitectureVersionV2

const (
	DevClusterGenevaLoggingAccount       = "AROClusterLogs"
	DevClusterGenevaLoggingConfigVersion = "2.4"
	DevClusterGenevaLoggingNamespace     = "AROClusterLogs"
	DevClusterGenevaMetricsAccount       = "AzureRedHatOpenShiftCluster"
	DevGenevaLoggingEnvironment          = "Test"
	DevRPGenevaLoggingAccount            = "ARORPLogs"
	DevRPGenevaLoggingConfigVersion      = "4.3"
	DevRPGenevaLoggingNamespace          = "ARORPLogs"
	DevRPGenevaMetricsAccount            = "AzureRedHatOpenShiftRP"

	DevGatewayGenevaLoggingConfigVersion = "4.3"
)

var GitCommit = "unknown"

// DefaultMinorVersion describes the minor OpenShift version to default to
var DefaultMinorVersion = 12

// DefaultInstallStreams describes the latest version of our supported streams
var DefaultInstallStreams = map[int]*Stream{
	10: {
		Version:  NewVersion(4, 10, 63),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:340091aefa0bba06bbb99cc58cb1f2b73404c832f72b83c526b8e7677efbecef",
	},
	11: {
		Version:  NewVersion(4, 11, 44),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:52cbfbbeb9cc03b49c2788ac7333e63d3dae14673e01a9d8e59270f3a8390ed3",
	},
	12: {
		Version:  NewVersion(4, 12, 17),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:7ca5f8aa44bbc537c5a985a523d87365eab3f6e72abc50b7be4caae741e093f4",
	},
}

// DefaultInstallStream describes stream we are defaulting to for all new clusters
var DefaultInstallStream = DefaultInstallStreams[DefaultMinorVersion]

var HiveInstallStreams = []*Stream{
	DefaultInstallStreams[10],
	{
		Version:  NewVersion(4, 10, 54),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:7e44fa5f6aa15f9492341c4714bba4dc5089c968f2bf77fb8d4cdf189634f8f5",
	},
	{
		Version:  NewVersion(4, 10, 40),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:b9fad814fb4442e7e852b0614d9bb4e2ebc5e1a2fa51623aa838b4ee0e4a5369",
	},
	DefaultInstallStreams[11],
	{
		Version:  NewVersion(4, 11, 26),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:1c3913a65b0a10b4a0650f54e545fe928360a94767acea64c0bd10faa52c945a",
	},
	DefaultInstallStreams[12],
	{
		Version:  NewVersion(4, 12, 24),
		PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:84a7b73f5393c8225c42edf214c82a8d00037a087222c5548ad1ed1b35cedb0e",
	},
}

// UpgradeStreams describes list of streams we support for upgrades
var (
	UpgradeStreams = []*Stream{
		DefaultInstallStream,
		{
			Version:  NewVersion(4, 9, 28),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:4084d94969b186e20189649b5affba7da59f7d1943e4e5bc7ef78b981eafb7a8",
		},
		{
			Version:  NewVersion(4, 8, 18),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:321aae3d3748c589bc2011062cee9fd14e106f258807dc2d84ced3f7461160ea",
		},
		{
			Version:  NewVersion(4, 7, 30),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:aba54b293dc151f5c0fd96d4353ced6ced3e7da6620c1c10714ab32d0577486f",
		},
		{
			Version:  NewVersion(4, 6, 44),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:d042aa235b538721a39989b13d7d9d3537af9b57e9fd10f485dd04461932ec85",
		},
		{
			Version:  NewVersion(4, 5, 39),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:c4b9eb565c64df97afe7841bbcc0469daec7973e46ae588739cc30ea9062172b",
		},
		{
			Version:  NewVersion(4, 4, 33),
			PullSpec: "quay.io/openshift-release-dev/ocp-release@sha256:a035dddd8a5e5c99484138951ef4aba021799b77eb9046f683a5466c23717738",
		},
	}
)

// FluentbitImage contains the location of the Fluentbit container image
func FluentbitImage(acrDomain string) string {
	return acrDomain + "/fluentbit:1.9.10-cm20230621"
}

// MdmImage contains the location of the MDM container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdmImage(acrDomain string) string {
	return acrDomain + "/genevamdm:2.2023.609.2051-821f47-20230706t0953"
}

// MdsdImage contains the location of the MDSD container image
// https://eng.ms/docs/products/geneva/collect/references/linuxcontainers
func MdsdImage(acrDomain string) string {
	return acrDomain + "/genevamdsd:mariner_20230706.2"
}

// MUOImage contains the location of the Managed Upgrade Operator container image
func MUOImage(acrDomain string) string {
	return acrDomain + "/managed-upgrade-operator:aro-b4"
}

// GateKeeperImage contains the location of the GateKeeper container image
func GateKeeperImage(acrDomain string) string {
	return acrDomain + "/gatekeeper:v3.11.1"
}
