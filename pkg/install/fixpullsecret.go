package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

func (i *Installer) fixPullSecret(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ps, err := i.kubernetescli.CoreV1().Secrets("openshift-config").Get("pull-secret", metav1.GetOptions{})
		if err != nil {
			return err
		}

		dockerconfig := string(ps.Data[".dockerconfigjson"])
		for _, rp := range i.doc.OpenShiftCluster.Properties.RegistryProfiles {
			if strings.Contains(rp.Name, ".azurecr.io") {
				dockerconfig, err = pullsecret.SetRegistryProfiles(dockerconfig, rp)
				if err != nil {
					return err
				}
			}
		}
		ps.Data[".dockerconfigjson"] = []byte(dockerconfig)

		_, err = i.kubernetescli.CoreV1().Secrets("openshift-config").Update(ps)
		return err
	})
}
