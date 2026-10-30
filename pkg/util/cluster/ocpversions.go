package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	pkgmirror "github.com/Azure/ARO-RP/pkg/mirror"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Get the OpenShift stream data for a given version string using the same
// source that pkg/mirror does (the OpenShift releases graph).
func getStreamFromOpenShiftReleases(ctx context.Context, versionString string) (version.Stream, error) {
	allVersions, err := pkgmirror.AddFromGraph(ctx, version.NewVersion(4, 19))
	if err != nil {
		return version.Stream{}, err
	}

	for _, node := range allVersions {
		if node.Version == versionString {
			p, err := version.ParseVersion(node.Version)
			if err != nil {
				return version.Stream{}, err
			}

			return version.Stream{
				Version:  p,
				PullSpec: node.Payload,
			}, nil
		}
	}

	return version.Stream{}, errors.New("Version not found in version graph")
}
