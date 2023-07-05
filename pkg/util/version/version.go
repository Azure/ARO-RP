package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var rxVersion = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(.*)`)

type Version struct {
	V      [3]uint32
	Suffix string
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d%s", v.V[0], v.V[1], v.V[2], v.Suffix)
}

func (v *Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

func NewVersion(vs ...uint32) *Version {
	v := &Version{}

	copy(v.V[:], vs)

	return v
}

func ParseVersion(vsn string) (*Version, error) {
	m := rxVersion.FindStringSubmatch(strings.TrimSpace(vsn))
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

func (v *Version) Eq(w *Version) bool {
	for i := 0; i < 3; i++ {
		if v.V[i] != w.V[i] {
			return false
		}
	}
	return v.Suffix == w.Suffix
}

func (v *Version) MinorVersion() string {
	return fmt.Sprintf("%d.%d", v.V[0], v.V[1])
}
