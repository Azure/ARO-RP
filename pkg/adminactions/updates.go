package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (a *adminactions) DisableUpdates(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := a.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		cv.Spec.Upstream = ""
		cv.Spec.Channel = ""

		_, err = a.configcli.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}

// ClusterUpgrade posts the new version and image to the cluster-version-operator
// which will effect the upgrade.
func (a *adminactions) ClusterUpgrade(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := a.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		cv.Spec.DesiredUpdate = &configv1.Update{
			Version: version.OpenShiftVersion,
			Image:   version.OpenShiftPullSpec,
		}

		_, err = a.configcli.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}
