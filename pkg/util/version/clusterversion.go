package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetClusterVersion fetches the version of the openshift cluster.
// Note that it assumes the most recently applied version is
// cv.Status.History[0] assuming the State == Completed.
// If for some reason there is no cluster version history, it will
// return the most recently updated version in history
func GetClusterVersion(cv *configv1.ClusterVersion) (*Version, error) {
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

func ClusterVersionIsLessThan4_4(ctx context.Context, configcli configclient.Interface) (bool, error) {
	cv, err := configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	v, err := GetClusterVersion(cv)
	if err != nil {
		return false, err
	}

	// 4.3 uses SRV records for etcd
	return v.Lt(NewVersion(4, 4)), nil
}
