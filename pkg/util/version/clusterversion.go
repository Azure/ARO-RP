package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"

	configv1 "github.com/openshift/api/config/v1"
)

func GetClusterVersion(cv *configv1.ClusterVersion) (*Version, error) {
	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return ParseVersion(history.Version)
		}
	}

	return nil, errors.New("unknown cluster version")
}

// GetDesiredVersion retrieves the version that a cluster is upgrading to, or if
// it is not upgrading, the existing cluster version.
func GetDesiredVersion(cv *configv1.ClusterVersion) (*Version, error) {
	if cv.Spec.DesiredUpdate != nil &&
		cv.Spec.DesiredUpdate.Version != "" {
		return ParseVersion(cv.Spec.DesiredUpdate.Version)
	}

	if cv.Status.Desired.Version != "" {
		return ParseVersion(cv.Status.Desired.Version)
	}

	return GetClusterVersion(cv)
}
