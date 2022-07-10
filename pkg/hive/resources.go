package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/azure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterDeploymentName      = "cluster"
	kubesecretName             = "admin-kube-secret"
	servicePrincipalSecretname = "serviceprincipal-secret"
)

func kubeAdminSecret(namespace string, kubeConfig []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubesecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"kubeconfig": kubeConfig,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func servicePrincipalSecret(namespace string, secret []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      servicePrincipalSecretname,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"osServicePrincipal.json": secret,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func clusterDeployment(namespace string, clusterName string, clusterID string, infraID string, location string) *hivev1.ClusterDeployment {
	return &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterDeploymentName,
			Namespace: namespace,
		},
		Spec: hivev1.ClusterDeploymentSpec{
			BaseDomain:  "",
			ClusterName: clusterName,
			Installed:   true,
			ClusterMetadata: &hivev1.ClusterMetadata{
				AdminKubeconfigSecretRef: corev1.LocalObjectReference{
					Name: kubesecretName,
				},
				ClusterID: clusterID,
				InfraID:   infraID,
			},
			Platform: hivev1.Platform{
				Azure: &azure.Platform{
					BaseDomainResourceGroupName: "",
					Region:                      location,
					CredentialsSecretRef: corev1.LocalObjectReference{
						Name: servicePrincipalSecretname,
					},
				},
			},
			PreserveOnDelete: true,
			ManageDNS:        false,
		},
	}
}
