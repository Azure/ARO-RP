package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
)

// generateAROServiceKubeconfig generates additional admin credentials and a
// kubeconfig for the ARO service, based on the admin kubeconfig found in the
// graph.
func (m *manager) generateAROServiceKubeconfig(pg persistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:aro-service", []string{"system:masters"})
}

// generateAROSREKubeconfig generates additional admin credentials and a
// kubeconfig for ARO SREs, based on the admin kubeconfig found in the graph.
func (m *manager) generateAROSREKubeconfig(pg persistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:aro-sre", nil)
}

func generateKubeconfig(pg persistedGraph, commonName string, organization []string) (*kubeconfig.AdminInternalClient, error) {
	var ca *tls.AdminKubeConfigSignerCertKey
	var adminInternalClient *kubeconfig.AdminInternalClient
	err := pg.get(&ca, &adminInternalClient)
	if err != nil {
		return nil, err
	}

	cfg := &tls.CertCfg{
		Subject:      pkix.Name{CommonName: commonName, Organization: organization},
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Validity:     tls.ValidityTenYears,
	}

	var clientCertKey tls.AdminKubeConfigClientCertKey

	err = clientCertKey.SignedCertKey.Generate(cfg, ca, strings.ReplaceAll(commonName, ":", "-"), tls.DoNotAppendParent)
	if err != nil {
		return nil, err
	}

	// create a Config for the new service kubeconfig based on the generated cluster admin Config
	aroInternalClient := kubeconfig.AdminInternalClient{}
	aroInternalClient.Config = &clientcmd.Config{
		Clusters: adminInternalClient.Config.Clusters,
		AuthInfos: []clientcmd.NamedAuthInfo{
			{
				Name: commonName,
				AuthInfo: clientcmd.AuthInfo{
					ClientCertificateData: clientCertKey.CertRaw,
					ClientKeyData:         clientCertKey.KeyRaw,
				},
			},
		},
		Contexts: []clientcmd.NamedContext{
			{
				Name: commonName,
				Context: clientcmd.Context{
					Cluster:  adminInternalClient.Config.Contexts[0].Context.Cluster,
					AuthInfo: commonName,
				},
			},
		},
		CurrentContext: commonName,
	}

	data, err := yaml.Marshal(aroInternalClient.Config)
	if err != nil {
		return nil, err
	}

	aroInternalClient.File = &asset.File{
		Filename: "auth/aro/kubeconfig",
		Data:     data,
	}

	return &aroInternalClient, nil
}
