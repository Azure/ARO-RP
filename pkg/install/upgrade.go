package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (i *Installer) upgradeCluster(ctx context.Context) error {
	vsn, err := version.ParseVersion(version.OpenShiftVersion)
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		desired, err := version.ParseVersion(cv.Status.Desired.Version)
		if err != nil {
			return err
		}

		if !desired.Lt(vsn) {
			i.log.Printf("not upgrading: cvo desired version is %s", cv.Status.Desired.Version)
			return nil
		}

		i.log.Printf("initiating cluster upgrade, target version %s", version.OpenShiftVersion)

		cv.Spec.DesiredUpdate = &configv1.Update{
			Version: version.OpenShiftVersion,
			Image:   version.OpenShiftPullSpec,
		}

		_, err = i.configcli.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}
