package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
)

// generateAROServiceKubeconfig generates additional admin credentials and kubeconfig
// based on admin kubeconfig found in graph
func (i *Installer) generateAROServiceKubeconfig(g graph) (*kubeconfig.AdminInternalClient, error) {
	ca := g[reflect.TypeOf(&tls.AdminKubeConfigSignerCertKey{})].(*tls.AdminKubeConfigSignerCertKey)
	cfg := &tls.CertCfg{
		Subject:      pkix.Name{CommonName: "system:aro-service", Organization: []string{"system:masters"}},
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Validity:     tls.ValidityTenYears,
	}

	var clientCertKey tls.AdminKubeConfigClientCertKey

	err := clientCertKey.SignedCertKey.Generate(cfg, ca, "system-aro-service", tls.DoNotAppendParent)
	if err != nil {
		return nil, err
	}

	// create a Config for the new service kubeconfig based on the generated cluster admin Config
	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)
	aroServiceInternalClient := kubeconfig.AdminInternalClient{}
	aroServiceInternalClient.Config = &clientcmd.Config{
		Clusters: adminInternalClient.Config.Clusters,
		AuthInfos: []clientcmd.NamedAuthInfo{
			{
				Name: "system:aro-service",
				AuthInfo: clientcmd.AuthInfo{
					ClientCertificateData: clientCertKey.CertRaw,
					ClientKeyData:         clientCertKey.KeyRaw,
				},
			},
		},
		Contexts: []clientcmd.NamedContext{
			{
				Name: "system:aro-service",
				Context: clientcmd.Context{
					Cluster:  adminInternalClient.Config.Contexts[0].Context.Cluster,
					AuthInfo: "system:aro-service",
				},
			},
		},
		CurrentContext: "system:aro-service",
	}

	data, err := yaml.Marshal(aroServiceInternalClient.Config)
	if err != nil {
		return nil, err
	}

	aroServiceInternalClient.File = &asset.File{
		Filename: "auth/aro/kubeconfig",
		Data:     data,
	}

	return &aroServiceInternalClient, nil
}
