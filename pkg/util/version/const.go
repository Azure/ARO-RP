package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	OpenShiftVersion    = "4.3.18"
	OpenShiftPullSpec   = "quay.io/openshift-release-dev/ocp-release@sha256:1f0fd38ac0640646ab8e7fec6821c8928341ad93ac5ca3a48c513ab1fb63bc4b"
	OpenShiftMustGather = "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:2e10ad0fc17f39c7a83aac32a725c78d7dd39cd9bbe3ec5ca0b76dcaa98416fa"
	GitCommitUnknown    = "unknown"
)

// GitCommit is a variable so it can be set in the Makefile, but logically a const.
var GitCommit = GitCommitUnknown
