package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

// if the cluster is using a managed domain and has a DigiCert-issued
// certificate, replace the certificate with one issued by OneCert. This
// ensures that clusters upgrading to 4.16 aren't blocked due to the SHA-1
// signing algorithm in use by DigiCert
func (m *manager) replaceDigicert(ctx context.Context) error {
	if strings.Contains(m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain, ".") {
		oneCertIssuerName := "OneCertV2-PublicCA"

		for _, certName := range []string{m.doc.ID + "-apiserver", m.doc.ID + "-ingress"} {
			clusterKeyvault := m.env.ClusterKeyvault()

			bundle, err := clusterKeyvault.GetCertificate(ctx, certName)
			if err != nil {
				return err
			}

			if strings.Contains(*bundle.Policy.IssuerParameters.Name, "DigiCert") {
				policy, err := clusterKeyvault.GetCertificatePolicy(ctx, certName)
				if err != nil {
					return err
				}

				policy.IssuerParameters.Name = &oneCertIssuerName
				err = clusterKeyvault.UpdateCertificatePolicy(ctx, certName, policy)
				if err != nil {
					return err
				}

				m.env.ClusterKeyvault().CreateSignedCertificate(ctx, oneCertIssuerName, certName, certName, keyvault.EkuServerAuth)
			}
		}
	}

	return nil
}
