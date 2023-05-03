package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

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

	tokenManager, err := acrtoken.NewManager(m.env, m.localFpAuthorizer)
	if err != nil {
		return err
	}

	rp := tokenManager.GetRegistryProfile(m.doc.OpenShiftCluster)
	if rp == nil {
		// shouldn't get here but return error if we do
		return nil // TODO: add some error
	}

	err = tokenManager.RotateTokenPassword(ctx, rp)
	if err != nil {
		return err
	}
	// update cluster pull secret in openshift-azure-operator namespace
	// secret is stored as a .dockerconfigjson string in the .dockerconfigjson key
	encodedDockerConfigJson, _, err := pullsecret.SetRegistryProfiles("", rp)
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

	return nil
}
