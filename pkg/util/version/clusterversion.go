package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
)

// GetClusterVersion fetches the version of the openshift cluster.
// Note that it assumes the most recently applied version is
// cv.Status.History[0] assuming the State == Completed.
// If for some reason there is no cluster version history, it will
// return the most recently updated version in history
func GetClusterVersion(cv *configv1.ClusterVersion) (Version, error) {
	unknownErr := errors.New("unknown cluster version")
	if cv == nil {
		return nil, unknownErr
	}

	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return ParseVersion(history.Version)
		}
	}

	// If the cluster history has no completed version, we're most likely installing
	// so grab the first history version and use it even if it's not completed
	if len(cv.Status.History) > 0 {
		return ParseVersion(cv.Status.History[0].Version)
	}

	return nil, unknownErr
}

// ClusterVersionLessThan checks if the current cluster version is less than the specified version
func ClusterVersionLessThan(ctx context.Context, configcli configclient.Interface, checkVersion Version) (bool, Version, error) {
	cv, err := configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return false, nil, err
	}
	clusterVersion, err := GetClusterVersion(cv)
	if err != nil {
		return false, clusterVersion, err
	}

	return clusterVersion.Lt(checkVersion), clusterVersion, nil
}

func IsClusterUpgrading(cv *configv1.ClusterVersion) bool {
	var isUpgrading bool
	if c := findClusterOperatorStatusCondition(cv.Status.Conditions, configv1.OperatorProgressing); c != nil && c.Status == configv1.ConditionTrue {
		isUpgrading = true
	} else {
		isUpgrading = false
	}
	return isUpgrading
}

func findClusterOperatorStatusCondition(conditions []configv1.ClusterOperatorStatusCondition, name configv1.ClusterStatusConditionType) *configv1.ClusterOperatorStatusCondition {
	for i := range conditions {
		if conditions[i].Type == name {
			return &conditions[i]
		}
	}
	return nil
}
