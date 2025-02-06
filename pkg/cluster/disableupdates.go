package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/coreos/go-semver/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	configv1 "github.com/openshift/api/config/v1"
)

func (m *manager) disableUpdates(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		cv.Spec.Upstream = ""
		cv.Spec.Channel = ""

		// when installing via Hive we replace the quay.io domain with the ACR domain, so now we set it back to the expected domain
		if m.installViaHive {
			version, err := m.openShiftVersionFromVersion(ctx)
			if err != nil {
				return err
			}
			parsedVersion, err := semver.NewVersion(version.Properties.Version)
			if err != nil {
				return err
			}
			parsedVersion.Metadata = ""
			cv.Spec.DesiredUpdate = &configv1.Update{
				Version: parsedVersion.String(),
				Image:   strings.Replace(version.Properties.OpenShiftPullspec, m.env.ACRDomain(), "quay.io", 1),
			}
		}

		_, err = m.configcli.ConfigV1().ClusterVersions().Update(ctx, cv, metav1.UpdateOptions{})
		return err
	})
}
