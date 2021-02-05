package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/status"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (k *kubeActions) Upgrade(ctx context.Context, upgradeY bool) error {
	return upgrade(ctx, k.log, k.configcli, version.Streams, upgradeY)
}

func upgrade(ctx context.Context, log *logrus.Entry, configcli configclient.Interface, streams []*version.Stream, upgradeY bool) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// We don't upgrade if cluster upgrade is not finished
		if !status.ClusterVersionOperatorIsHealthy(cv.Status) {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Not upgrading: cvo is unhealthy.")
		}

		desired, err := version.ParseVersion(cv.Status.Desired.Version)
		if err != nil {
			return err
		}

		// Get Cluster upgrade version based on desired version
		stream := version.GetUpgradeStream(streams, desired, upgradeY)
		if stream == nil {
			log.Info("not upgrading: stream not found")
			return nil
		}

		log.Printf("initiating cluster upgrade, target version %s", stream.Version.String())

		cv.Spec.DesiredUpdate = &configv1.Update{
			Version: stream.Version.String(),
			Image:   stream.PullSpec,
		}

		_, err = configcli.ConfigV1().ClusterVersions().Update(ctx, cv, metav1.UpdateOptions{})
		return err
	})
}
