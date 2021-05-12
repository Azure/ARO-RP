package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/cluster/graph"
)

// generateAROServiceKubeconfig generates additional admin credentials and a
// kubeconfig for the ARO service, based on the admin kubeconfig found in the
// graph.
func (m *manager) generateAROServiceKubeconfig(pg graph.PersistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:aro-service", []string{"system:masters"}, tls.ValidityTenYears)
}

// generateAROSREKubeconfig generates additional admin credentials and a
// kubeconfig for ARO SREs, based on the admin kubeconfig found in the graph.
func (m *manager) generateAROSREKubeconfig(pg graph.PersistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:aro-sre", nil, tls.ValidityTenYears)
}

// generateUserAdminKubeconfig generates additional admin credentials and a
// kubeconfig for ARO User, based on the admin kubeconfig found in the graph.
func (m *manager) generateUserAdminKubeconfig(pg graph.PersistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:admin", nil, tls.ValidityOneYear)
}

func generateKubeconfig(pg graph.PersistedGraph, commonName string, organization []string, validity time.Duration) (*kubeconfig.AdminInternalClient, error) {
	var ca *tls.AdminKubeConfigSignerCertKey
	var adminInternalClient *kubeconfig.AdminInternalClient
	err := pg.Get(&ca, &adminInternalClient)
	if err != nil {
		return nil, err
	}

	cfg := &tls.CertCfg{
		Subject:      pkix.Name{CommonName: commonName, Organization: organization},
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Validity:     validity,
	}

	var clientCertKey tls.AdminKubeConfigClientCertKey

	err = clientCertKey.SignedCertKey.Generate(cfg, ca, strings.ReplaceAll(commonName, ":", "-"), tls.DoNotAppendParent)
	if err != nil {
		return nil, err
	}

	// create a Config for the new service kubeconfig based on the generated cluster admin Config
	aroInternalClient := kubeconfig.AdminInternalClient{}
	aroInternalClient.Config = &clientcmdv1.Config{
		Clusters: adminInternalClient.Config.Clusters,
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: commonName,
				AuthInfo: clientcmdv1.AuthInfo{
					ClientCertificateData: clientCertKey.CertRaw,
					ClientKeyData:         clientCertKey.KeyRaw,
				},
			},
		},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: commonName,
				Context: clientcmdv1.Context{
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
