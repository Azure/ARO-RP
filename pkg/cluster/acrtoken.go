package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	err = m.rotateOpenShiftConfigSecret(ctx, pullSecret.Data[corev1.DockerConfigJsonKey])
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) rotateOpenShiftConfigSecret(ctx context.Context, encodedDockerConfigJson []byte) error {
	openshiftConfigSecret, err := m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Get(ctx, pullSecretName.Name, metav1.GetOptions{})
	if err != nil {
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
		err = m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Delete(ctx, pullSecretName.Name, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return err
		}
		_, err = m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Create(ctx, recreatedSecret, metav1.CreateOptions{})
		return err
	}

	openshiftConfigSecret.Data[corev1.DockerConfigJsonKey] = encodedDockerConfigJson
	_, err = m.kubernetescli.CoreV1().Secrets(pullSecretName.Namespace).Update(ctx, openshiftConfigSecret, metav1.UpdateOptions{})

	return err
}
