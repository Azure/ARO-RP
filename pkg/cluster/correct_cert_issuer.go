package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
)

// if the cluster isn't using a managed domain and has a DigiCert-issued
// certificate, replace the certificate with one issued by OneCert. This
// ensures that clusters upgrading to 4.16 aren't blocked due to the SHA-1
// signing algorithm in use by DigiCert
func (m *manager) correctCertificateIssuer(ctx context.Context) error {
	if !strings.Contains(m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain, ".") {
		apiCertName := m.doc.ID + "-apiserver"
		apiHostname := strings.Split(strings.TrimPrefix(m.doc.OpenShiftCluster.Properties.APIServerProfile.URL, "https://"), ":")[0]
		err := m.ensureCertificateIssuer(ctx, apiCertName, apiHostname, "OneCertV2-PublicCA")
		if err != nil {
			return err
		}

		ingressCertName := m.doc.ID + "-ingress"
		ingressHostname := "*" + strings.TrimSuffix(strings.TrimPrefix(m.doc.OpenShiftCluster.Properties.ConsoleProfile.URL, "https://console-openshift-console"), "/")
		err = m.ensureCertificateIssuer(ctx, ingressCertName, ingressHostname, "OneCertV2-PublicCA")
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) ensureCertificateIssuer(ctx context.Context, certificateName, dnsName, issuerName string) error {
	if strings.Count(dnsName, ".") < 2 {
		return errors.New(fmt.Sprintf("%s is not a valid DNS name", dnsName))
	}

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

		err = clusterKeyvault.CreateSignedCertificate(ctx, issuerName, certificateName, dnsName, keyvault.EkuServerAuth)
		if err != nil {
			return err
		}
	}
	return nil
}
