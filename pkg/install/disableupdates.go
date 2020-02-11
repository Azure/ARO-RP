package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

func (i *Installer) disableUpdates(ctx context.Context) error {
	restConfig, err := restconfig.RestConfig(ctx, i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}
	cli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := cli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		cv.Spec.Upstream = ""
		cv.Spec.Channel = ""

		_, err = cli.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}
