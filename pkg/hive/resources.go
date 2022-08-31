package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivev1azure "github.com/openshift/hive/apis/hive/v1/azure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

// Changing values of these constants most likely would require
// some sort of migration on the Hive cluster for existing clusters.
const (
	ClusterDeploymentName             = "cluster"
	aroServiceKubeconfigSecretName    = "aro-service-kubeconfig-secret"
	clusterServicePrincipalSecretName = "cluster-service-principal-secret"
)

var (
	devEnvVars = []string{
		"AZURE_FP_CLIENT_ID",
		"AZURE_RP_CLIENT_ID",
		"AZURE_RP_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"DOMAIN_NAME",
		"KEYVAULT_PREFIX",
		"LOCATION",
		"PROXY_HOSTNAME",
		"PULL_SECRET",
		"RESOURCEGROUP",
	}
	prodEnvVars = []string{
		"AZURE_FP_CLIENT_ID",
		"CLUSTER_MDSD_ACCOUNT",
		"CLUSTER_MDSD_CONFIG_VERSION",
		"CLUSTER_MDSD_NAMESPACE",
		"DOMAIN_NAME",
		"GATEWAY_DOMAINS",
		"GATEWAY_RESOURCEGROUP",
		"KEYVAULT_PREFIX",
		"MDSD_ENVIRONMENT",
	}
)

func (hr *clusterManager) resources(sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) ([]kruntime.Object, error) {
	namespace := doc.OpenShiftCluster.Properties.HiveProfile.Namespace
	clusterSP, err := clusterSPToBytes(sub, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	cd := adoptedClusterDeployment(
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

func envSecret(namespace string, isDevelopment bool) *corev1.Secret {
	stringdata := map[string]string{}

	if isDevelopment {
		for _, i := range devEnvVars {
			stringdata["ARO_"+i] = os.Getenv(i)
		}
	} else {
		for _, i := range prodEnvVars {
			stringdata["ARO_"+i] = os.Getenv(i)
		}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      envSecretsName,
		},
		StringData: stringdata,
	}
}

func adoptedClusterDeployment(namespace, clusterName, clusterID, infraID, location, APIServerPrivateEndpointIP string) *hivev1.ClusterDeployment {
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

func pullsecretSecret(namespace string, oc *api.OpenShiftCluster) (*corev1.Secret, error) {
	pullSecret, err := pullsecret.Build(oc, string(oc.Properties.ClusterProfile.PullSecret))
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"cloud.openshift.com"} {
		pullSecret, err = pullsecret.RemoveKey(pullSecret, key)
		if err != nil {
			return nil, err
		}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      pullsecretSecretName,
		},
		StringData: map[string]string{
			".dockerconfigjson": pullSecret,
		},
	}, nil
}

func installConfigCM(namespace string, location string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      installConfigName,
		},
		StringData: map[string]string{
			"install-config.yaml": fmt.Sprintf(installConfigTemplate, location),
		},
	}
}
