package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Upgrade struct {
	Version    *Version
	PullSpec   string
	MustGather string
	Latest     bool
}

var (
	Upgrades = []Upgrade{
		{
			Version:    NewVersion(4, 3, 18),
			PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:1f0fd38ac0640646ab8e7fec6821c8928341ad93ac5ca3a48c513ab1fb63bc4b",
			MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:2e10ad0fc17f39c7a83aac32a725c78d7dd39cd9bbe3ec5ca0b76dcaa98416fa",
		},
		{
			Version:    NewVersion(4, 4, 10),
			PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:0d1ffca302ae55d32574b38438c148d33c2a8a05c8daf97eeb13e9ab948174f7",
			MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:06ae5b7e36f23eb2e5ae5826499978de4e0124e33938c2e532ed73563b1f7c14",
			Latest:     true,
		},
	}
)
