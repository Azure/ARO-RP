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

func GetClusterVersion(cv *configv1.ClusterVersion) (*Version, error) {
	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return ParseVersion(history.Version)
		}
	}

	return nil, errors.New("unknown cluster version")
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
