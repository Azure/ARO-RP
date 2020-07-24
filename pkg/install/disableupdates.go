package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (i *Installer) disableUpdates(ctx context.Context, configClient configclient.Interface) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := configClient.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		cv.Spec.Upstream = ""
		cv.Spec.Channel = ""

		_, err = configClient.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}
