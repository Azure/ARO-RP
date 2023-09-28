package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

func (m *manager) ensureACRToken(ctx context.Context) error {
	if m.env.IsLocalDevelopmentMode() {
		return nil
	}

	token, err := acrtoken.NewManager(m.env, m.localFpAuthorizer)
	if err != nil {
		return err
	}

	rp := token.GetRegistryProfile(m.doc.OpenShiftCluster)
	if rp == nil {
		// 1. choose a name and establish the intent to create a token with
		// that name
		rp = token.NewRegistryProfile(m.doc.OpenShiftCluster)

		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			token.PutRegistryProfile(doc.OpenShiftCluster, rp)
			return nil
		})
		if err != nil {
			return err
		}
	}

	if rp.Password == "" {
		// 2. ensure a token with the chosen name exists, generate a
		// password for it and store it in the database
		password, err := token.EnsureTokenAndPassword(ctx, rp)
		if err != nil {
			return err
		}

		rp.Password = api.SecureString(password)

		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			token.PutRegistryProfile(doc.OpenShiftCluster, rp)
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) rotateACRTokenPassword(ctx context.Context) error {
	// we do not want to rotate tokens in local development
	if m.env.IsLocalDevelopmentMode() || m.env.IsCI() {
		return nil
	}

	token, err := acrtoken.NewManager(m.env, m.localFpAuthorizer)
	if err != nil {
		return err
	}

	registryProfile := token.GetRegistryProfile(m.doc.OpenShiftCluster)
	err = token.RotateTokenPassword(ctx, registryProfile)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		token.PutRegistryProfile(doc.OpenShiftCluster, registryProfile)
		return nil
	})
	if err != nil {
		return err
	}

	// update cluster pull secret in openshift-azure-operator namespace
	// secret is stored as a .dockerconfigjson string in the .dockerconfigjson key
	encodedDockerConfigJson, _, err := pullsecret.SetRegistryProfiles("", registryProfile)
	if err != nil {
		return err
	}

	// wait for response from operator that reconciliation is completed successfully
	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operator.SecretName,
			Namespace: operator.Namespace,
		},
		Data: make(map[string][]byte),
	}
	pullSecret.Data[corev1.DockerConfigJsonKey] = []byte(encodedDockerConfigJson)

	_, err = m.kubernetescli.CoreV1().Secrets(operator.Namespace).Update(ctx, pullSecret, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	err = retryOperation(func() error {
		return m.rotateOpenShiftConfigSecret(ctx, pullSecret.Data[corev1.DockerConfigJsonKey])
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) rotateOpenShiftConfigSecret(ctx context.Context, encodedDockerConfigJson []byte) error {
	openshiftConfigSecret, err := m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Get(ctx, pullSecretName.Name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	recreationOfSecretRequired := openshiftConfigSecret == nil ||
		(openshiftConfigSecret.Type != corev1.SecretTypeDockerConfigJson || openshiftConfigSecret.Data == nil) ||
		(openshiftConfigSecret.Immutable != nil && *openshiftConfigSecret.Immutable)

	if recreationOfSecretRequired {
		recreatedSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pullSecretName.Name,
				Namespace: pullSecretName.Namespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{corev1.DockerConfigJsonKey: encodedDockerConfigJson},
		}

		err := retryOperation(func() error {
			return m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Delete(ctx, pullSecretName.Name, metav1.DeleteOptions{})
		})
		if err != nil && !kerrors.IsNotFound(err) {
			return err
		}

		// attempt to merge if we can, defaults to the created pull secret
		if openshiftConfigSecret != nil && openshiftConfigSecret.Data != nil {
			previousConfigData, previousConfigDataExists := openshiftConfigSecret.Data[corev1.DockerConfigJsonKey]
			if previousConfigDataExists {
				mergedPullSecretData, _, err := pullsecret.Merge(string(previousConfigData), string(encodedDockerConfigJson))
				if err == nil {
					recreatedSecret.Data[corev1.DockerConfigJsonKey] = []byte(mergedPullSecretData)
				} else {
					m.log.Error("Could not merge openshift config pull secret, overriding with new acr token", err)
				}
			}
		}
		return retryOperation(func() error {
			_, err = m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Create(ctx, recreatedSecret, metav1.CreateOptions{})
			return err
		})
	}

	// update flow
	mergedPullSecretData, _, err := pullsecret.Merge(string(openshiftConfigSecret.Data[corev1.DockerConfigJsonKey]), string(encodedDockerConfigJson))
	if err == nil {
		openshiftConfigSecret.Data[corev1.DockerConfigJsonKey] = []byte(mergedPullSecretData)
	} else {
		openshiftConfigSecret.Data[corev1.DockerConfigJsonKey] = encodedDockerConfigJson
	}

	return retryOperation(func() error {
		_, err = m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Update(ctx, openshiftConfigSecret, metav1.UpdateOptions{})
		return err
	})
}

func retryOperation(retryable func() error) error {
	return retry.OnError(wait.Backoff{
		Steps:    10,
		Duration: 2 * time.Second,
	}, func(err error) bool {
		return kerrors.IsBadRequest(err) || kerrors.IsInternalError(err) || kerrors.IsServerTimeout(err) || kerrors.IsConflict(err)
	}, retryable)
}
