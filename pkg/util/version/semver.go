package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/coreos/go-semver/semver"
)

// CreateSemverFromMinorVersionString takes in a string representing a semantic version number that is
// missing the patch version from the end (ex.: "4.13") and appends a ".0" and returns a semver.Version.
// It results in a panic if v + ".0" does not turn out to be a valid semantic version number. This function
// is useful for applications such as making it easier to compare strings that represent OpenShift minor
// versions.
func CreateSemverFromMinorVersionString(v string) *semver.Version {
	return semver.New(v + ".0")
}
