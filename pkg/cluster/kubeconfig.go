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
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// generateAROServiceKubeconfig generates additional admin credentials and a
// kubeconfig for the ARO service, based on the admin kubeconfig found in the
// graph.
func (m *manager) generateAROServiceKubeconfig(pg graph.PersistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:aro-service", []string{"system:masters"}, tls.ValidityTenYears, true)
}

// generateAROSREKubeconfig generates additional admin credentials and a
// kubeconfig for ARO SREs, based on the admin kubeconfig found in the graph.
func (m *manager) generateAROSREKubeconfig(pg graph.PersistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:aro-sre", nil, tls.ValidityTenYears, true)
}

// checkUserAdminKubeconfigUpdated checks if the user kubeconfig is
// present, has >275days until expiry, has the right settings
func (m *manager) checkUserAdminKubeconfigUpdated() bool {
	if len(m.doc.OpenShiftCluster.Properties.UserAdminKubeconfig) == 0 {
		// field empty, not updated
		return false
	}
	var userAdminKubeconfig clientcmdv1.Config
	err := yaml.Unmarshal([]byte(m.doc.OpenShiftCluster.Properties.UserAdminKubeconfig), &userAdminKubeconfig)
	if err != nil {
		// yaml invalid, not updated
		return false
	}
	for i := range userAdminKubeconfig.Clusters {
		if strings.HasPrefix(userAdminKubeconfig.Clusters[i].Cluster.Server, "https://api-int.") {
			// URL pointing to api-int, not updated
			// TODO remove this after PUCM has been run on all clusters.
			return false
		}
	}
	for i := range userAdminKubeconfig.AuthInfos {
		var b []byte
		b = append(b, userAdminKubeconfig.AuthInfos[i].AuthInfo.ClientCertificateData...)
		b = append(b, userAdminKubeconfig.AuthInfos[i].AuthInfo.ClientKeyData...)
		innerkey, innercert, err := utilpem.Parse(b)
		if err != nil {
			// error while parsing cert or key, not updated
			return false
		}
		if innerkey == nil {
			// no client key, not updated
			return false
		}
		for j := range innercert {
			if !innercert[j].NotAfter.After(time.Now().AddDate(0, 0, 275)) {
				// Not After field in certificate closer than 275 days, not updated
				return false
			}
		}
	}

	// passed all checks, it's up to date
	return true
}

// generateUserAdminKubeconfig generates additional admin credentials and a
// kubeconfig for ARO User, based on the admin kubeconfig found in the graph.
func (m *manager) generateUserAdminKubeconfig(pg graph.PersistedGraph) (*kubeconfig.AdminInternalClient, error) {
	return generateKubeconfig(pg, "system:admin", nil, tls.ValidityOneYear, false)
}

func (m *manager) generateKubeconfigs(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
	if err != nil {
		return err
	}

	var adminInternalClient *kubeconfig.AdminInternalClient
	err = pg.Get(true, &adminInternalClient)
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
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminInternalClient.File.Data
		doc.OpenShiftCluster.Properties.AROServiceKubeconfig = aroServiceInternalClient.File.Data
		doc.OpenShiftCluster.Properties.AROSREKubeconfig = aroSREInternalClient.File.Data
		doc.OpenShiftCluster.Properties.UserAdminKubeconfig = aroUserInternalClient.File.Data
		return nil
	})
	return err
}

func generateKubeconfig(pg graph.PersistedGraph, commonName string, organization []string, validity time.Duration, internal bool) (*kubeconfig.AdminInternalClient, error) {
	var ca *tls.AdminKubeConfigSignerCertKey
	var adminInternalClient *kubeconfig.AdminInternalClient
	err := pg.Get(true, &ca, &adminInternalClient)
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

	if !internal {
		for i := range aroInternalClient.Config.Clusters {
			// user kubeconfig should point to external URL not api-int, which has a properly signed cert
			aroInternalClient.Config.Clusters[i].Cluster.Server = strings.Replace(aroInternalClient.Config.Clusters[i].Cluster.Server, "https://api-int.", "https://api.", 1)
			aroInternalClient.Config.Clusters[i].Cluster.CertificateAuthorityData = nil
		}
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
