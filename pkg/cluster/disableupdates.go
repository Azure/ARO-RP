package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (m *manager) disableUpdates(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		cv.Spec.Upstream = ""
		cv.Spec.Channel = ""

		// when installing via Hive we end up with an ACR image pullspec and we should leave it for the customer as quay.io
		if m.installViaHive {
			version, err := m.openShiftVersionFromVersion(ctx)
			if err != nil {
				return err
			}
			cv.Spec.DesiredUpdate = &configv1.Update{
				Version: version.Properties.Version,
				Image:   strings.Replace(version.Properties.OpenShiftPullspec, m.env.ACRDomain(), "quay.io", 1),
			}
		}

		_, err = m.configcli.ConfigV1().ClusterVersions().Update(ctx, cv, metav1.UpdateOptions{})
		return err
	})
}
