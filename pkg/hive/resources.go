package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/yaml"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivev1azure "github.com/openshift/hive/apis/hive/v1/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

// Changing values of these constants most likely would require
// some sort of migration on the Hive cluster for existing clusters.
const (
	ClusterDeploymentName                   = "cluster"
	aroServiceKubeconfigSecretName          = "aro-service-kubeconfig-secret"
	clusterServicePrincipalSecretName       = "cluster-service-principal-secret"
	clusterManifestsSecretName              = "cluster-manifests-secret"
	boundServiceAccountSigningKeySecretName = "bound-service-account-signing-key"
	boundServiceAccountSigningKeySecretKey  = "bound-service-account-signing-key.key"
	hiveClusterPlatformLabel                = "hive.openshift.io/cluster-platform"
	hiveClusterRegionLabel                  = "hive.openshift.io/cluster-region"
	hiveInfraDisabledAnnotation             = "hive.openshift.io/infra-disabled"
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
		"ACR_RESOURCE_ID",
	}
)

func (hr *clusterManager) resources(sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) ([]kruntime.Object, error) {
	namespace := doc.OpenShiftCluster.Properties.HiveProfile.Namespace

	cd := adoptedClusterDeployment(
		namespace,
		doc.OpenShiftCluster.Name,
		doc.ID,
		doc.OpenShiftCluster.Properties.InfraID,
		doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
		doc.OpenShiftCluster.Location,
		doc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP,
		doc.OpenShiftCluster.Properties.ClusterProfile.Domain,
	)
	err := utillog.EnrichHiveWithCorrelationData(cd, doc.CorrelationData)
	if err != nil {
		return nil, err
	}
	err = utillog.EnrichHiveWithResourceID(cd, doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	azureCredentialSecret, err := clusterAzureSecret(namespace, doc.OpenShiftCluster, sub)
	if err != nil {
		return nil, err
	}

	return []kruntime.Object{
		aroServiceKubeconfigSecret(namespace, doc.OpenShiftCluster.Properties.AROServiceKubeconfig),
		azureCredentialSecret,
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

func clusterAzureSecret(namespace string, oc *api.OpenShiftCluster, sub *api.SubscriptionDocument) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterServicePrincipalSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	// Add osServicePrincipal.json only when cluster is not managed identity
	if !oc.UsesWorkloadIdentity() {
		clusterSPBytes, err := clusterSPToBytes(sub, oc)
		if err != nil {
			return nil, err
		}
		secret.Data["osServicePrincipal.json"] = clusterSPBytes
	}

	return secret, nil
}

func clusterManifestsSecret(namespace string, customManifests map[string]kruntime.Object) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterManifestsSecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{},
		Type:       corev1.SecretTypeOpaque,
	}

	for key, manifest := range customManifests {
		b, err := yaml.Marshal(manifest)
		if err != nil {
			return nil, err
		}

		secret.StringData[key] = string(b)
	}
	return secret, nil
}

func boundSASigningKeySecret(namespace string, oc *api.OpenShiftCluster) (*corev1.Secret, error) {
	if !oc.UsesWorkloadIdentity() {
		// no secret required - Hive ClusterDeployment should not reference this secret
		return nil, nil
	}
	if oc.Properties.ClusterProfile.BoundServiceAccountSigningKey == nil {
		return nil, fmt.Errorf("properties.clusterProfile.boundServiceAccountSigningKey not set")
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      boundServiceAccountSigningKeySecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			boundServiceAccountSigningKeySecretKey: string(*oc.Properties.ClusterProfile.BoundServiceAccountSigningKey),
		},
		Type: corev1.SecretTypeOpaque,
	}, nil
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

func adoptedClusterDeployment(namespace, clusterName, clusterID, infraID, resourceGroupID, location, APIServerPrivateEndpointIP, clusterDomain string) *hivev1.ClusterDeployment {
	if !strings.ContainsRune(clusterDomain, '.') {
		clusterDomain += "." + os.Getenv("DOMAIN_NAME")
	}
	return &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterDeploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				hiveClusterPlatformLabel: "azure",
				hiveClusterRegionLabel:   location,
			},
			Annotations: map[string]string{
				// https://github.com/openshift/hive/pull/2501
				// Disable hibernation controller as it is not used as part of ARO's Hive implementation
				hiveInfraDisabledAnnotation: "true",
			},
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
					BaseDomainResourceGroupName: resourceGroupID,
					Region:                      location,
					CredentialsSecretRef: corev1.LocalObjectReference{
						Name: clusterServicePrincipalSecretName,
					},
				},
			},
			ControlPlaneConfig: hivev1.ControlPlaneConfigSpec{
				APIServerIPOverride: APIServerPrivateEndpointIP,
				APIURLOverride:      fmt.Sprintf("api-int.%s:6443", clusterDomain),
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
