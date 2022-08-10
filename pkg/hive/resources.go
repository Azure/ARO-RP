package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivev1azure "github.com/openshift/hive/apis/hive/v1/azure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

// Changing values of these constants most likely would require
// some sort of migration on the Hive cluster for existing clusters.
const (
	ClusterDeploymentName             = "cluster"
	aroServiceKubeconfigSecretName    = "aro-service-kubeconfig-secret"
	clusterServicePrincipalSecretName = "cluster-service-principal-secret"
)

func (hr *clusterManager) resources(sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) ([]kruntime.Object, error) {
	namespace := doc.OpenShiftCluster.Properties.HiveProfile.Namespace
	clusterSP, err := clusterSPToBytes(sub, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	cd := clusterDeployment(
		namespace,
		doc.OpenShiftCluster.Name,
		doc.ID,
		doc.OpenShiftCluster.Properties.InfraID,
		doc.OpenShiftCluster.Location,
		doc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP,
	)
	err = utillog.EnrichHiveWithCorrelationData(cd, doc.CorrelationData)
	if err != nil {
		return nil, err
	}
	err = utillog.EnrichHiveWithResourceID(cd, doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	return []kruntime.Object{
		aroServiceKubeconfigSecret(namespace, doc.OpenShiftCluster.Properties.AROServiceKubeconfig),
		clusterServicePrincipalSecret(namespace, clusterSP),
		cd,
	}, nil
}

func aroServiceKubeconfigSecret(namespace string, kubeConfig []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aroServiceKubeconfigSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"kubeconfig": kubeConfig,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func clusterServicePrincipalSecret(namespace string, secret []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterServicePrincipalSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"osServicePrincipal.json": secret,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func clusterDeployment(namespace, clusterName, clusterID, infraID, location, APIServerPrivateEndpointIP string) *hivev1.ClusterDeployment {
	return &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterDeploymentName,
			Namespace: namespace,
		},
		Spec: hivev1.ClusterDeploymentSpec{
			BaseDomain:  "",
			ClusterName: clusterName,
			Installed:   true,
			ClusterMetadata: &hivev1.ClusterMetadata{
				AdminKubeconfigSecretRef: corev1.LocalObjectReference{
					Name: aroServiceKubeconfigSecretName,
				},
				ClusterID: clusterID,
				InfraID:   infraID,
			},
			Platform: hivev1.Platform{
				Azure: &hivev1azure.Platform{
					BaseDomainResourceGroupName: "",
					Region:                      location,
					CredentialsSecretRef: corev1.LocalObjectReference{
						Name: clusterServicePrincipalSecretName,
					},
				},
			},
			ControlPlaneConfig: hivev1.ControlPlaneConfigSpec{
				APIServerIPOverride: APIServerPrivateEndpointIP,
			},
			PreserveOnDelete: true,
			ManageDNS:        false,
		},
	}
}
