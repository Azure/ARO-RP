package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/tls"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
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

func (m *manager) generateKubeconfigs(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	var adminInternalClient *kubeconfig.AdminInternalClient
	err = pg.Get(&adminInternalClient)
	if err != nil {
		return err
	}

	aroServiceInternalClient, err := m.generateAROServiceKubeconfig(pg)
	if err != nil {
		return err
	}
	aroSREInternalClient, err := m.generateAROSREKubeconfig(pg)
	if err != nil {
		return err
	}
	aroUserInternalClient, err := m.generateUserAdminKubeconfig(pg)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		// used for the SAS token with which the bootstrap node retrieves its
		// ignition payload
		var t time.Time
		if doc.OpenShiftCluster.Properties.Install.Now == t {
			// Only set this if it hasn't been set already, since it is used to
			// create values for signedStart and signedExpiry in
			// deployResourceTemplate, and if these are not stable a
			// redeployment will fail.
			doc.OpenShiftCluster.Properties.Install.Now = time.Now().UTC()
		}
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminInternalClient.File.Data
		doc.OpenShiftCluster.Properties.AROServiceKubeconfig = aroServiceInternalClient.File.Data
		doc.OpenShiftCluster.Properties.AROSREKubeconfig = aroSREInternalClient.File.Data
		doc.OpenShiftCluster.Properties.UserAdminKubeconfig = aroUserInternalClient.File.Data
		return nil
	})
	return err
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
