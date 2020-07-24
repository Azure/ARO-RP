package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

func (i *Installer) fixPullSecret(ctx context.Context, kubernetesClient kubernetes.Interface) error {
	// TODO: this function does not currently reapply a pull secret in
	// development mode.

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var isCreate bool
		ps, err := kubernetesClient.CoreV1().Secrets("openshift-config").Get("pull-secret", metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			ps = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeDockerConfigJson,
			}
			isCreate = true
		case err != nil:
			return err
		}

		if ps.Data == nil {
			ps.Data = map[string][]byte{}
		}

		if !json.Valid(ps.Data[v1.DockerConfigJsonKey]) {
			delete(ps.Data, v1.DockerConfigJsonKey)
		}

		pullSecret, changed, err := pullsecret.SetRegistryProfiles(string(ps.Data[v1.DockerConfigJsonKey]), i.doc.OpenShiftCluster.Properties.RegistryProfiles...)
		if err != nil {
			return err
		}

		if ps.Type != v1.SecretTypeDockerConfigJson {
			ps = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret",
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{},
			}
			isCreate = true
			changed = true

			// unfortunately the type field is immutable.
			err = kubernetesClient.CoreV1().Secrets(ps.Namespace).Delete(ps.Name, nil)
			if err != nil {
				return err
			}

			// there is a small risk of crashing here: if that happens, we will
			// restart, create a new pull secret, and will have dropped the rest
			// of the customer's pull secret on the floor :-(
		}

		if !changed {
			return nil
		}

		ps.Data[v1.DockerConfigJsonKey] = []byte(pullSecret)

		if isCreate {
			_, err = kubernetesClient.CoreV1().Secrets(ps.Namespace).Create(ps)
		} else {
			_, err = kubernetesClient.CoreV1().Secrets(ps.Namespace).Update(ps)
		}
		return err
	})
}
