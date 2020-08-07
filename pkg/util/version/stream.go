package version

import (
	"fmt"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Stream struct {
	Version    *Version
	PullSpec   string
	MustGather string
}

// GetUpgradeStream determines if a valid upgrade path is available, and if so, returns the corresponding stream.
func GetUpgradeStream(v *Version) (*Stream, error) {
	// ARO version matches OCP version - X.Y.Z.
	// We know we can have only single version configured per Y release. These
	// version should be edge points for Y+1 upgrades.

	// We check first with which configured Y stream we are dealing with
	for _, upgradeCandidate := range Streams {
		if upgradeCandidate.Version.V[0] == v.V[0] &&
			upgradeCandidate.Version.V[1] == v.V[1] {

			// we DO NOT upgrade if CVO version is already higher
			if upgradeCandidate.Version != nil && upgradeCandidate.Version.Lt(v) {
				return nil, fmt.Errorf("not upgrading: cvo desired version is %s", v)
			}

			// If upgradeCandidate is higher than CVO - use it to upgrade to ARO
			// latest x.Y release before jumping major version.
			if v.Lt(upgradeCandidate.Version) {
				return &upgradeCandidate, nil
			}

			// If we on right version, we need to upgrade next major version
			if v.Eq(upgradeCandidate.Version) {
				for _, upgradeCandidate := range Streams {
					// if incoming version is 4.2, we return 4.3 for major upgrade.
					if upgradeCandidate.Version.V[1] == v.V[1]+1 {
						return &upgradeCandidate, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("not upgrading: stream not found %s", v)
}
