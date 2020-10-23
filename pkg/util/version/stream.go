package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Stream struct {
	Version  *Version
	PullSpec string
}

// GetUpgradeStream returns an upgrade Stream for a Version or nil if no upgrade
// should be performed.
func GetUpgradeStream(v *Version, upgradeY bool) *Stream {
	s := getStream(v)
	if s == nil {
		return nil
	}

	if v.Lt(s.Version) {
		return s
	}

	if upgradeY {
		return getStream(&Version{V: [3]uint32{v.V[0], v.V[1] + 1}})
	}

	return nil
}

// getStream receives a Version x.y.z and returns the Stream x.y.0 if it exists.
func getStream(v *Version) *Stream {
	for _, s := range Streams {
		if s.Version.V[0] == v.V[0] && s.Version.V[1] == v.V[1] {
			return s
		}
	}

	return nil
}
