package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"reflect"
	"time"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmd "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

// generateAndStoreKubeconfigs generates additional admin credentials and kubeconfig based admin kubeconfig
// found in graph and stores both to the database
func (i *Installer) generateAndStoreKubeconfigs(ctx context.Context, g graph, aroServiceName string) error {

	ca := g[reflect.TypeOf(&tls.AdminKubeConfigSignerCertKey{})].(*tls.AdminKubeConfigSignerCertKey)
	clientCertKey := g[reflect.TypeOf(&tls.AdminKubeConfigClientCertKey{})].(*tls.AdminKubeConfigClientCertKey)
	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)

	cfg := &tls.CertCfg{
		Subject:      pkix.Name{CommonName: aroServiceName, Organization: []string{"system:masters"}},
		KeyUsages:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Validity:     tls.ValidityTenYears,
	}

	// new key and certificate will be stored in k.KeyRaw and k.CertRaw
	err := clientCertKey.SignedCertKey.Generate(cfg, ca, "admin-kubeconfig-client", tls.DoNotAppendParent)
	if err != nil {
		return err
	}

	// create a Config for the new service kubeconfig based on the generated cluster admin Config
	clusters := make([]clientcmd.NamedCluster, len(adminInternalClient.Config.Clusters))
	copy(clusters, adminInternalClient.Config.Clusters)

	aroServiceConfig := clientcmd.Config{
		Preferences: adminInternalClient.Config.Preferences,
		Clusters:    clusters,
		AuthInfos: []clientcmd.NamedAuthInfo{
			clientcmd.NamedAuthInfo{
				Name: aroServiceName,
				AuthInfo: clientcmd.AuthInfo{
					ClientCertificateData: clientCertKey.CertRaw,
					ClientKeyData:         clientCertKey.KeyRaw,
				},
			},
		},
		Contexts: []clientcmd.NamedContext{
			clientcmd.NamedContext{
				Name: aroServiceName,
				Context: clientcmd.Context{
					Cluster:  adminInternalClient.Config.Contexts[0].Context.Cluster,
					AuthInfo: aroServiceName,
				},
			},
		},
		CurrentContext: aroServiceName,
	}

	data, err := yaml.Marshal(aroServiceConfig)
	if err != nil {
		return err
	}

	var aroServiceInternalClient kubeconfig.AdminInternalClient
	aroServiceInternalClient.File = &asset.File{
		Filename: adminInternalClient.File.Filename, // need to change the name
		Data:     data,
	}
	aroServiceInternalClient.Config = &aroServiceConfig

	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		// used for the SAS token with which the bootstrap node retrieves its
		// ignition payload
		doc.OpenShiftCluster.Properties.Install.Now = time.Now().UTC()
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminInternalClient.File.Data
		doc.OpenShiftCluster.Properties.AroServiceKubeconfig = aroServiceInternalClient.File.Data
		return nil
	})

	return err
}
