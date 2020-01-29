package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/cluster"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/machines"
	"github.com/openshift/installer/pkg/asset/manifests"
	"github.com/openshift/installer/pkg/asset/password"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/rhcos"
	"github.com/openshift/installer/pkg/asset/templates/content/bootkube"
	"github.com/openshift/installer/pkg/asset/templates/content/openshift"
	"github.com/openshift/installer/pkg/asset/tls"
)

var registeredTypes = map[string]asset.Asset{
	"*bootkube.CVOOverrides":                                  &bootkube.CVOOverrides{},
	"*bootkube.EtcdCAConfigMap":                               &bootkube.EtcdCAConfigMap{},
	"*bootkube.EtcdClientSecret":                              &bootkube.EtcdClientSecret{},
	"*bootkube.EtcdHostService":                               &bootkube.EtcdHostService{},
	"*bootkube.EtcdHostServiceEndpoints":                      &bootkube.EtcdHostServiceEndpoints{},
	"*bootkube.EtcdMetricClientSecret":                        &bootkube.EtcdMetricClientSecret{},
	"*bootkube.EtcdMetricServingCAConfigMap":                  &bootkube.EtcdMetricServingCAConfigMap{},
	"*bootkube.EtcdMetricSignerSecret":                        &bootkube.EtcdMetricSignerSecret{},
	"*bootkube.EtcdNamespace":                                 &bootkube.EtcdNamespace{},
	"*bootkube.EtcdService":                                   &bootkube.EtcdService{},
	"*bootkube.EtcdServingCAConfigMap":                        &bootkube.EtcdServingCAConfigMap{},
	"*bootkube.EtcdSignerSecret":                              &bootkube.EtcdSignerSecret{},
	"*bootkube.KubeCloudConfig":                               &bootkube.KubeCloudConfig{},
	"*bootkube.KubeSystemConfigmapRootCA":                     &bootkube.KubeSystemConfigmapRootCA{},
	"*bootkube.MachineConfigServerTLSSecret":                  &bootkube.MachineConfigServerTLSSecret{},
	"*bootkube.OpenshiftConfigSecretPullSecret":               &bootkube.OpenshiftConfigSecretPullSecret{},
	"*bootkube.OpenshiftMachineConfigOperator":                &bootkube.OpenshiftMachineConfigOperator{},
	"*bootstrap.Bootstrap":                                    &bootstrap.Bootstrap{},
	"*cluster.Metadata":                                       &cluster.Metadata{},
	"*cluster.TerraformVariables":                             &cluster.TerraformVariables{},
	"*installconfig.ClusterID":                                &installconfig.ClusterID{},
	"*installconfig.InstallConfig":                            &installconfig.InstallConfig{},
	"*installconfig.PlatformCreds":                            &installconfig.PlatformCreds{},
	"*installconfig.PlatformCredsCheck":                       &installconfig.PlatformCredsCheck{},
	"*kubeconfig.AdminClient":                                 &kubeconfig.AdminClient{},
	"*kubeconfig.AdminInternalClient":                         &kubeconfig.AdminInternalClient{},
	"*kubeconfig.Kubelet":                                     &kubeconfig.Kubelet{},
	"*kubeconfig.LoopbackClient":                              &kubeconfig.LoopbackClient{},
	"*machine.Master":                                         &machine.Master{},
	"*machine.Worker":                                         &machine.Worker{},
	"*machines.Master":                                        &machines.Master{},
	"*machines.Worker":                                        &machines.Worker{},
	"*manifests.AdditionalTrustBundleConfig":                  &manifests.AdditionalTrustBundleConfig{},
	"*manifests.CloudProviderConfig":                          &manifests.CloudProviderConfig{},
	"*manifests.DNS":                                          &manifests.DNS{},
	"*manifests.ImageContentSourcePolicy":                     &manifests.ImageContentSourcePolicy{},
	"*manifests.Infrastructure":                               &manifests.Infrastructure{},
	"*manifests.Ingress":                                      &manifests.Ingress{},
	"*manifests.Manifests":                                    &manifests.Manifests{},
	"*manifests.Networking":                                   &manifests.Networking{},
	"*manifests.Openshift":                                    &manifests.Openshift{},
	"*manifests.Proxy":                                        &manifests.Proxy{},
	"*manifests.Scheduler":                                    &manifests.Scheduler{},
	"*openshift.CloudCredsSecret":                             &openshift.CloudCredsSecret{},
	"*openshift.KubeadminPasswordSecret":                      &openshift.KubeadminPasswordSecret{},
	"*openshift.NetworkCRDs":                                  &openshift.NetworkCRDs{},
	"*openshift.PrivateClusterOutbound":                       &openshift.PrivateClusterOutbound{},
	"*openshift.RoleCloudCredsSecretReader":                   &openshift.RoleCloudCredsSecretReader{},
	"*password.KubeadminPassword":                             &password.KubeadminPassword{},
	"*releaseimage.Image":                                     &releaseimage.Image{},
	"*rhcos.BootstrapImage":                                   new(rhcos.BootstrapImage),
	"*rhcos.Image":                                            new(rhcos.Image),
	"*tls.AdminKubeConfigCABundle":                            &tls.AdminKubeConfigCABundle{},
	"*tls.AdminKubeConfigClientCertKey":                       &tls.AdminKubeConfigClientCertKey{},
	"*tls.AdminKubeConfigSignerCertKey":                       &tls.AdminKubeConfigSignerCertKey{},
	"*tls.AggregatorCA":                                       &tls.AggregatorCA{},
	"*tls.AggregatorCABundle":                                 &tls.AggregatorCABundle{},
	"*tls.AggregatorClientCertKey":                            &tls.AggregatorClientCertKey{},
	"*tls.AggregatorSignerCertKey":                            &tls.AggregatorSignerCertKey{},
	"*tls.APIServerProxyCertKey":                              &tls.APIServerProxyCertKey{},
	"*tls.EtcdCABundle":                                       &tls.EtcdCABundle{},
	"*tls.EtcdMetricCABundle":                                 &tls.EtcdMetricCABundle{},
	"*tls.EtcdMetricSignerCertKey":                            &tls.EtcdMetricSignerCertKey{},
	"*tls.EtcdMetricSignerClientCertKey":                      &tls.EtcdMetricSignerClientCertKey{},
	"*tls.EtcdSignerCertKey":                                  &tls.EtcdSignerCertKey{},
	"*tls.EtcdSignerClientCertKey":                            &tls.EtcdSignerClientCertKey{},
	"*tls.JournalCertKey":                                     &tls.JournalCertKey{},
	"*tls.KubeAPIServerCompleteCABundle":                      &tls.KubeAPIServerCompleteCABundle{},
	"*tls.KubeAPIServerCompleteClientCABundle":                &tls.KubeAPIServerCompleteClientCABundle{},
	"*tls.KubeAPIServerExternalLBServerCertKey":               &tls.KubeAPIServerExternalLBServerCertKey{},
	"*tls.KubeAPIServerInternalLBServerCertKey":               &tls.KubeAPIServerInternalLBServerCertKey{},
	"*tls.KubeAPIServerLBCABundle":                            &tls.KubeAPIServerLBCABundle{},
	"*tls.KubeAPIServerLBSignerCertKey":                       &tls.KubeAPIServerLBSignerCertKey{},
	"*tls.KubeAPIServerLocalhostCABundle":                     &tls.KubeAPIServerLocalhostCABundle{},
	"*tls.KubeAPIServerLocalhostServerCertKey":                &tls.KubeAPIServerLocalhostServerCertKey{},
	"*tls.KubeAPIServerLocalhostSignerCertKey":                &tls.KubeAPIServerLocalhostSignerCertKey{},
	"*tls.KubeAPIServerServiceNetworkCABundle":                &tls.KubeAPIServerServiceNetworkCABundle{},
	"*tls.KubeAPIServerServiceNetworkServerCertKey":           &tls.KubeAPIServerServiceNetworkServerCertKey{},
	"*tls.KubeAPIServerServiceNetworkSignerCertKey":           &tls.KubeAPIServerServiceNetworkSignerCertKey{},
	"*tls.KubeAPIServerToKubeletCABundle":                     &tls.KubeAPIServerToKubeletCABundle{},
	"*tls.KubeAPIServerToKubeletClientCertKey":                &tls.KubeAPIServerToKubeletClientCertKey{},
	"*tls.KubeAPIServerToKubeletSignerCertKey":                &tls.KubeAPIServerToKubeletSignerCertKey{},
	"*tls.KubeControlPlaneCABundle":                           &tls.KubeControlPlaneCABundle{},
	"*tls.KubeControlPlaneKubeControllerManagerClientCertKey": &tls.KubeControlPlaneKubeControllerManagerClientCertKey{},
	"*tls.KubeControlPlaneKubeSchedulerClientCertKey":         &tls.KubeControlPlaneKubeSchedulerClientCertKey{},
	"*tls.KubeControlPlaneSignerCertKey":                      &tls.KubeControlPlaneSignerCertKey{},
	"*tls.KubeletBootstrapCABundle":                           &tls.KubeletBootstrapCABundle{},
	"*tls.KubeletBootstrapCertSigner":                         &tls.KubeletBootstrapCertSigner{},
	"*tls.KubeletClientCABundle":                              &tls.KubeletClientCABundle{},
	"*tls.KubeletClientCertKey":                               &tls.KubeletClientCertKey{},
	"*tls.KubeletCSRSignerCertKey":                            &tls.KubeletCSRSignerCertKey{},
	"*tls.KubeletServingCABundle":                             &tls.KubeletServingCABundle{},
	"*tls.MCSCertKey":                                         &tls.MCSCertKey{},
	"*tls.RootCA":                                             &tls.RootCA{},
	"*tls.ServiceAccountKeyPair":                              &tls.ServiceAccountKeyPair{},
}

type graph map[reflect.Type]asset.Asset

func (g graph) resolve(a asset.Asset) (asset.Asset, error) {
	if _, found := g[reflect.TypeOf(a)]; !found {
		for _, dep := range a.Dependencies() {
			_, err := g.resolve(dep)
			if err != nil {
				return nil, err
			}
		}

		err := a.Generate(asset.Parents(g))
		if err != nil {
			return nil, err
		}

		g[reflect.TypeOf(a)] = a
	}

	return g[reflect.TypeOf(a)], nil
}
