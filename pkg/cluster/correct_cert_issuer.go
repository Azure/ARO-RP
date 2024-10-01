package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

// if the cluster isn't using a managed domain and has a DigiCert-issued
// certificate, replace the certificate with one issued by OneCert. This
// ensures that clusters upgrading to 4.16 aren't blocked due to the SHA-1
// signing algorithm in use by DigiCert
func (m *manager) correctCertificateIssuer(ctx context.Context) error {
	if !strings.Contains(m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain, ".") {
		for _, certName := range []string{m.doc.ID + "-apiserver", m.doc.ID + "-ingress"} {
			err := m.ensureCertificateIssuer(ctx, certName, "OneCertV2-PublicCA")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *manager) ensureCertificateIssuer(ctx context.Context, certificateName, issuerName string) error {
	clusterKeyvault := m.env.ClusterKeyvault()

	bundle, err := clusterKeyvault.GetCertificate(ctx, certificateName)
	if err != nil {
		return err
	}

	if *bundle.Policy.IssuerParameters.Name != issuerName {
		policy, err := clusterKeyvault.GetCertificatePolicy(ctx, certificateName)
		if err != nil {
			return err
		}

		policy.IssuerParameters.Name = &issuerName
		err = clusterKeyvault.UpdateCertificatePolicy(ctx, certificateName, policy)
		if err != nil {
			return err
		}

		err = clusterKeyvault.CreateSignedCertificate(ctx, issuerName, certificateName, certificateName, keyvault.EkuServerAuth)
		if err != nil {
			return err
		}
	}
	return nil
}
