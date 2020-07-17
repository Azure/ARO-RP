package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"regexp"
	"strconv"
)

var rxVersion = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(.*)`)

type Version struct {
	V      [3]uint32
	Suffix string
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.V[0], v.V[1], v.V[2])
}

func NewVersion(vs ...uint32) *Version {
	v := &Version{}

	copy(v.V[:], vs)

	return v
}

func ParseVersion(vsn string) (*Version, error) {
	m := rxVersion.FindStringSubmatch(vsn)
	if m == nil {
		return nil, fmt.Errorf("could not parse version %q", vsn)
	}

	v := &Version{
		Suffix: m[4],
	}

	for i := 0; i < 3; i++ {
		d, err := strconv.ParseUint(m[i+1], 10, 32)
		if err != nil {
			return nil, err
		}

		v.V[i] = uint32(d)
	}

	return v, nil
}

func (v *Version) Lt(w *Version) bool {
	for i := 0; i < 3; i++ {
		switch {
		case v.V[i] < w.V[i]:
			return true
		case v.V[i] > w.V[i]:
			return false
		}
	}

	return false
}
