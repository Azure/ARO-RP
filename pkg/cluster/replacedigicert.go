package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

// if the cluster is using a managed domain and has a DigiCert-issued
// certificate, replace the certificate with one issued by OneCert. This
// ensures that clusters upgrading to 4.16 aren't blocked due to the SHA-1
// signing algorithm in use by DigiCert
func (m *manager) replaceDigicert(ctx context.Context) error {
	apiCertName := m.doc.ID + "apiserver"

	if strings.Contains(m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain, ".") {
		bundle, err := m.env.ClusterKeyvault().GetSecret(ctx, apiCertName)
		if err != nil {
			return err
		}

		// don't need to look at the key, just the cert(s)
		_, certs, err := utilpem.Parse([]byte(*bundle.Value))
		if err != nil {
			return err
		}

	outer:
		for _, cert := range certs {
			for _, w := range cert.Issuer.Organization {
				if strings.Contains(w, "DigiCert") {
					// cluster uses a DigiCert certificate, change it over to OneCert
					_, err := m.env.ClusterKeyvault().SetCertificateIssuer(ctx, "OneCertV2-PublicCA", azkeyvault.CertificateIssuerSetParameters{})
					if err != nil {
						return err
					}

					m.env.ClusterKeyvault().CreateSignedCertificate(ctx, "OneCertV2-PublicCA", apiCertName, cert.Subject.CommonName, keyvault.EkuServerAuth)
					break outer
				}
			}
		}
	}

	return nil
}
