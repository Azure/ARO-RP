package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
)

func GetClusterVersion(cv *configv1.ClusterVersion) (*Version, error) {
	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return ParseVersion(history.Version)
		}
	}

	return nil, errors.New("unknown cluster version")
}

func ClusterVersionIsGreaterThan4_3(ctx context.Context, configcli configclient.Interface, logEntry *logrus.Entry) bool {
	v, err := GetClusterVersion(ctx, configcli)
	if err != nil {
		logEntry.Print(err)
		return false
	}

	if v.Lt(NewVersion(4, 4)) {
		// 4.3 uses SRV records for etcd
		logEntry.Printf("cluster version < 4.4, not removing private DNS zone")
		return false
	}
	return true
}
