package version

import "fmt"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Stream struct {
	Version    *Version
	PullSpec   string
	MustGather string
}

// GetStream return matching stream, used to upgrade cluster.
func GetStream(v *Version) (*Stream, error) {
	for _, s := range Streams {
		if s.Version.V[0] == v.V[0] &&
			s.Version.V[1] == v.V[1] {

			// we DO NOT upgrade if CVO version is already higher
			if !v.Lt(s.Version) {
				return nil, fmt.Errorf("not upgrading: cvo desired version is %s", v)
			}

			return &s, nil
		}
	}
	return nil, fmt.Errorf("stream for %s not found", v)
}
