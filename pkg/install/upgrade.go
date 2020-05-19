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
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		desired, err := version.ParseVersion(cv.Status.Desired.Version)
		if err != nil {
			return err
		}

		// Get Cluster upgrade version based on desired version
		// If desired is 4.3.x we return 4.3 channel update
		// If desired is 4.4.x we return 4.4 channel update
		stream, err := version.GetStream(desired)
		if err != nil {
			i.log.Info(err)
			return nil
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
