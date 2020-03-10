package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"
)

// generateAndStoreKubeconfigs generates additional admin credentials and kubeconfig based admin kubeconfig
// found in graph and stores both to the database
func (i *Installer) generateAROServiceKubeconfig(ctx context.Context, g graph, aroServiceName string, aroServiceInternalClient *kubeconfig.AdminInternalClient) error {
	ca := g[reflect.TypeOf(&tls.AdminKubeConfigSignerCertKey{})].(*tls.AdminKubeConfigSignerCertKey)
	cfg := &tls.CertCfg{
		Subject:      pkix.Name{CommonName: aroServiceName, Organization: []string{"system:masters"}},
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Validity:     tls.ValidityTenYears,
	}

	var clientCertKey tls.AdminKubeConfigClientCertKey
	err := clientCertKey.SignedCertKey.Generate(cfg, ca, "admin-kubeconfig-client", tls.DoNotAppendParent)
	if err != nil {
		return err
	}

	// create a Config for the new service kubeconfig based on the generated cluster admin Config
	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)
	aroServiceInternalClient.Config = &clientcmd.Config{
		Preferences: adminInternalClient.Config.Preferences,
		Clusters:    adminInternalClient.Config.Clusters,
		AuthInfos: []clientcmd.NamedAuthInfo{
			{
				Name: aroServiceName,
				AuthInfo: clientcmd.AuthInfo{
					ClientCertificateData: clientCertKey.CertRaw,
					ClientKeyData:         clientCertKey.KeyRaw,
				},
			},
		},
		Contexts: []clientcmd.NamedContext{
			{
				Name: aroServiceName,
				Context: clientcmd.Context{
					Cluster:  adminInternalClient.Config.Contexts[0].Context.Cluster,
					AuthInfo: aroServiceName,
				},
			},
		},
		CurrentContext: aroServiceName,
	}

	data, err := yaml.Marshal(aroServiceInternalClient.Config)
	if err != nil {
		return err
	}

	aroServiceInternalClient.File = &asset.File{
		Filename: "auth/aro/kubeconfig",
		Data:     data,
	}

	return nil
}
