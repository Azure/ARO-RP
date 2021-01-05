package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"reflect"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/bootstraplogging"
	clusterAsset "github.com/openshift/installer/pkg/asset/cluster"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/machines"
	"github.com/openshift/installer/pkg/asset/manifests"
	"github.com/openshift/installer/pkg/asset/openshiftinstall"
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
	"*bootkube.AROWorkerRegistries":                           &bootkube.AROWorkerRegistries{},
	"*bootstrap.Bootstrap":                                    &bootstrap.Bootstrap{},
	"*cluster.Metadata":                                       &clusterAsset.Metadata{},
	"*bootstraplogging.Config":                                &bootstraplogging.Config{},
	"*installconfig.ClusterID":                                &installconfig.ClusterID{},
	"*installconfig.InstallConfig":                            &installconfig.InstallConfig{},
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
	"*openshift.BaremetalConfig":                              &openshift.BaremetalConfig{},
	"*openshift.CloudCredsSecret":                             &openshift.CloudCredsSecret{},
	"*openshift.KubeadminPasswordSecret":                      &openshift.KubeadminPasswordSecret{},
	"*openshift.NetworkCRDs":                                  &openshift.NetworkCRDs{},
	"*openshift.PrivateClusterOutbound":                       &openshift.PrivateClusterOutbound{},
	"*openshift.RoleCloudCredsSecretReader":                   &openshift.RoleCloudCredsSecretReader{},
	"*openshiftinstall.Config":                                &openshiftinstall.Config{},
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
	"*tls.BootstrapSSHKeyPair":                                &tls.BootstrapSSHKeyPair{},
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

// graph values are interface{}s rather than Assets to enable us to round trip
// values that we don't recognise (i.e. deprecated types from older installers)
type graph map[string]interface{}

func newGraph(assets ...asset.Asset) graph {
	g := graph{}
	for _, a := range assets {
		g.set(a)
	}
	return g
}

func (g graph) get(a asset.Asset) asset.Asset {
	i := g[reflect.TypeOf(a).String()]
	if i == nil {
		return nil
	}

	return i.(asset.Asset)
}

func (g graph) set(a asset.Asset) {
	g[reflect.TypeOf(a).String()] = a
}

func (g graph) resolve(a asset.Asset) error {
	if g.get(a) != nil {
		return nil
	}

	for _, dep := range a.Dependencies() {
		err := g.resolve(dep)
		if err != nil {
			return err
		}
	}

	parents := asset.Parents{}
	for _, v := range g {
		parents[reflect.TypeOf(v)] = v.(asset.Asset)
	}

	err := a.Generate(parents)
	if err != nil {
		return err
	}

	g.set(a)

	return nil
}

func (m *manager) graphExists(ctx context.Context) (bool, error) {
	m.log.Print("checking if graph exists")

	blobService, err := m.getBlobService(ctx, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return false, err
	}

	aro := blobService.GetContainerReference("aro")
	return aro.GetBlobReference("graph").Exists()
}

func (m *manager) loadGraph(ctx context.Context) (graph, error) {
	m.log.Print("load graph")

	blobService, err := m.getBlobService(ctx, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return nil, err
	}

	aro := blobService.GetContainerReference("aro")
	cluster := aro.GetBlobReference("graph")
	rc, err := cluster.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	b, err = m.aead.Open(b)
	if err != nil {
		return nil, err
	}

	var g graph
	err = json.Unmarshal(b, &g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (m *manager) saveGraph(ctx context.Context, g graph) error {
	m.log.Print("save graph")

	blobService, err := m.getBlobService(ctx, mgmtstorage.Permissions("cw"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return err
	}

	bootstrap := g.get(&bootstrap.Bootstrap{}).(*bootstrap.Bootstrap)
	bootstrapIgn := blobService.GetContainerReference("ignition").GetBlobReference("bootstrap.ign")
	err = bootstrapIgn.CreateBlockBlobFromReader(bytes.NewReader(bootstrap.File.Data), nil)
	if err != nil {
		return err
	}

	graph := blobService.GetContainerReference("aro").GetBlobReference("graph")
	b, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		return err
	}

	b, err = m.aead.Seal(b)
	if err != nil {
		return err
	}

	return graph.CreateBlockBlobFromReader(bytes.NewReader(b), nil)
}
