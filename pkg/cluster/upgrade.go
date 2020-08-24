package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/status"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (i *manager) upgradeCluster(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// We don't upgrade if cluster upgrade is not finished
		if !status.ClusterVersionOperatorIsHealthy(cv.Status) {
			return fmt.Errorf("not upgrading: previous upgrade in-progress")
		}

		desired, err := version.ParseVersion(cv.Status.Desired.Version)
		if err != nil {
			return err
		}

		// Get Cluster upgrade version based on desired version
		stream, err := version.GetUpgradeStream(desired)
		if err != nil {
			return err
		}

		i.log.Printf("initiating cluster upgrade, target version %s", stream.Version.String())

		cv.Spec.DesiredUpdate = &configv1.Update{
			Version: stream.Version.String(),
			Image:   stream.PullSpec,
		}

		_, err = i.configcli.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}
