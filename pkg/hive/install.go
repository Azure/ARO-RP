package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hivev1azure "github.com/openshift/hive/apis/hive/v1/azure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	createdByHiveLabelKey = "aro-created-by-Hive"
	envSecretsName        = "aro-env-secret"
	pullsecretSecretName  = "aro-pullsecret"
	installConfigName     = "aro-installconfig"
	installConfigTemplate = `apiVersion: v1
platform:
  azure:
    region: "%s"
`
)

func makeEnvSecret(name string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: "ARO_" + name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: envSecretsName,
				},
				Key: "ARO_" + name,
			},
		},
	}
}

func (c *clusterManager) Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error {
	sppSecret, err := servicePrincipalSecretForInstall(doc.OpenShiftCluster, sub, c.env.IsLocalDevelopmentMode())
	if err != nil {
		return err
	}

	psSecret, err := pullsecretSecret(doc.OpenShiftCluster.Properties.HiveProfile.Namespace, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	cd := c.clusterDeploymentForInstall(doc, version, c.env.IsLocalDevelopmentMode())

	// Enrich the cluster deployment with the correlation data so that logs are
	// properly annotated
	err = utillog.EnrichHiveWithCorrelationData(cd, doc.CorrelationData)
	if err != nil {
		return err
	}
	err = utillog.EnrichHiveWithResourceID(cd, doc.OpenShiftCluster.ID)
	if err != nil {
		return err
	}

	resources := []kruntime.Object{
		sppSecret,
		envSecret(doc.OpenShiftCluster.Properties.HiveProfile.Namespace, c.env.IsLocalDevelopmentMode()),
		psSecret,
		installConfigCM(doc.OpenShiftCluster.Properties.HiveProfile.Namespace, doc.OpenShiftCluster.Location),
		cd,
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	err = c.dh.Ensure(ctx, resources...)
	if err != nil {
		return err
	}

	return nil
}

func servicePrincipalSecretForInstall(oc *api.OpenShiftCluster, sub *api.SubscriptionDocument, isDevelopment bool) (*corev1.Secret, error) {
	clusterSPBytes, err := clusterSPToBytes(sub, oc)
	if err != nil {
		return nil, err
	}

	enc, err := json.Marshal(oc)
	if err != nil {
		return nil, err
	}

	encSub, err := json.Marshal(sub.Subscription)
	if err != nil {
		return nil, err
	}

	sppSecret := clusterServicePrincipalSecret(oc.Properties.HiveProfile.Namespace, clusterSPBytes)
	sppSecret.Data["99_aro.json"] = enc
	sppSecret.Data["99_sub.json"] = encSub

	if isDevelopment {
		// In development mode, load in the proxy certificates so that clusters
		// can be accessed from a local (not in Azure) Hive

		basepath := os.Getenv("ARO_CHECKOUT_PATH")
		if basepath == "" {
			// This assumes we are running from an ARO-RP checkout in development
			var err error
			_, curmod, _, _ := runtime.Caller(0)
			basepath, err = filepath.Abs(filepath.Join(filepath.Dir(curmod), "../.."))
			if err != nil {
				return nil, err
			}
		}

		proxyCert, err := os.ReadFile(path.Join(basepath, "secrets/proxy.crt"))
		if err != nil {
			return nil, err
		}

		proxyClientCert, err := os.ReadFile(path.Join(basepath, "secrets/proxy-client.crt"))
		if err != nil {
			return nil, err
		}

		proxyClientKey, err := os.ReadFile(path.Join(basepath, "secrets/proxy-client.key"))
		if err != nil {
			return nil, err
		}

		sppSecret.Data["proxy.crt"] = proxyCert
		sppSecret.Data["proxy-client.crt"] = proxyClientCert
		sppSecret.Data["proxy-client.key"] = proxyClientKey
	}

	return sppSecret, nil
}

func (c *clusterManager) clusterDeploymentForInstall(doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion, isDevelopment bool) *hivev1.ClusterDeployment {
	var envVars = []corev1.EnvVar{
		{
			Name:  "ARO_UUID",
			Value: doc.ID,
		},
	}

	if isDevelopment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "ARO_RP_MODE",
			Value: "development",
		})
		for _, i := range devEnvVars {
			envVars = append(envVars, makeEnvSecret(i))
		}
	} else {
		for _, i := range prodEnvVars {
			envVars = append(envVars, makeEnvSecret(i))
		}
	}

	clusterDomain := doc.OpenShiftCluster.Properties.ClusterProfile.Domain
	if !strings.ContainsRune(clusterDomain, '.') {
		clusterDomain += "." + os.Getenv("DOMAIN_NAME")
	}

	// Do not set InfraID here, as Hive wants to do that
	return &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterDeploymentName,
			Namespace: doc.OpenShiftCluster.Properties.HiveProfile.Namespace,
			Labels: map[string]string{
				"hive.openshift.io/cluster-platform": "azure",
				"hive.openshift.io/cluster-region":   doc.OpenShiftCluster.Location,
				createdByHiveLabelKey:                "true",
			},
			Annotations: map[string]string{
				"hive.openshift.io/try-install-once":                "true",
				"hive.openshift.io/cli-domain-from-installer-image": "true",
			},
		},
		Spec: hivev1.ClusterDeploymentSpec{
			BaseDomain:  "",
			ClusterName: doc.OpenShiftCluster.Name,
			Platform: hivev1.Platform{
				Azure: &hivev1azure.Platform{
					BaseDomainResourceGroupName: doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
					Region:                      doc.OpenShiftCluster.Location,
					CredentialsSecretRef: corev1.LocalObjectReference{
						Name: clusterServicePrincipalSecretName,
					},
				},
			},
			ControlPlaneConfig: hivev1.ControlPlaneConfigSpec{
				APIServerIPOverride: doc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP,
				APIURLOverride:      fmt.Sprintf("api-int.%s:6443", clusterDomain),
			},
			PullSecretRef: &corev1.LocalObjectReference{
				Name: pullsecretSecretName,
			},
			Provisioning: &hivev1.Provisioning{
				InstallerImageOverride: version.Properties.InstallerPullspec,
				ReleaseImage:           version.Properties.OpenShiftPullspec,
				InstallConfigSecretRef: &corev1.LocalObjectReference{
					Name: installConfigName,
				},
				InstallerEnv: envVars,
			},
			PreserveOnDelete: true,
			ManageDNS:        false,
		},
	}
}
