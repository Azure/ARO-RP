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

type VersionParseError struct {
	version string
}

func (e VersionParseError) Error() string {
	return fmt.Sprintf("could not parse version %q", e.version)
}

var rxVersion = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(.*)`)

type Version interface {
	Lt(Version) bool
	Gt(Version) bool
	Eq(Version) bool
	String() string
	MarshalJSON() ([]byte, error)
	Components() ([3]uint32, string)
	MinorVersion() string
}

type version struct {
	V      [3]uint32
	Suffix string
}

func (v *version) String() string {
	return fmt.Sprintf("%d.%d.%d%s", v.V[0], v.V[1], v.V[2], v.Suffix)
}

func (v *version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

func NewVersion(vs ...uint32) *version {
	v := &version{}

	copy(v.V[:], vs)

	return v
}

func ParseVersion(vsn string) (*version, error) {
	m := rxVersion.FindStringSubmatch(strings.TrimSpace(vsn))
	if m == nil {
		return nil, VersionParseError{version: vsn}
	}

	v := &version{
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

func (v *version) Lt(_w Version) bool {
	other, _ := _w.Components()
	for i := 0; i < 3; i++ {
		switch {
		case v.V[i] < other[i]:
			return true
		case v.V[i] > other[i]:
			return false
		}
	}

	return false
}

func (v *version) Gt(w Version) bool {
	return !v.Lt(w)
}

func (v *version) Eq(w Version) bool {
	return v.String() == w.String()
}

func (v *version) Components() ([3]uint32, string) {
	return v.V, v.Suffix
}

func (v *version) MinorVersion() string {
	return fmt.Sprintf("%d.%d", v.V[0], v.V[1])
}
