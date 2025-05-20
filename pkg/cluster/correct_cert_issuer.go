package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcertificates"
	"github.com/Azure/ARO-RP/pkg/util/dns"
)

// if the cluster is using a managed domain and has a DigiCert-issued
// certificate, replace the certificate with one issued by OneCert. This
// ensures that clusters upgrading to 4.16 aren't blocked due to the SHA-1
// signing algorithm in use by DigiCert
func (m *manager) correctCertificateIssuer(ctx context.Context) error {
	if m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		return nil
	}

	domain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
	if err != nil {
		return err
	}

	if domain != "" {
		apiHostname := strings.Split(strings.TrimPrefix(m.doc.OpenShiftCluster.Properties.APIServerProfile.URL, "https://"), ":")[0]
		err := m.ensureCertificateIssuer(ctx, m.APICertName(), apiHostname, OneCertPublicIssuerName)
		if err != nil {
			return err
		}

		ingressHostname := "*" + strings.TrimSuffix(strings.TrimPrefix(m.doc.OpenShiftCluster.Properties.ConsoleProfile.URL, "https://console-openshift-console"), "/")
		err = m.ensureCertificateIssuer(ctx, m.IngressCertName(), ingressHostname, OneCertPublicIssuerName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) ensureCertificateIssuer(ctx context.Context, certificateName, dnsName, issuerName string) error {
	if strings.Count(dnsName, ".") < 2 {
		return fmt.Errorf("%s is not a valid DNS name", dnsName)
	}

	clusterKeyvault := m.env.ClusterCertificates()

	bundle, err := clusterKeyvault.GetCertificate(ctx, certificateName, "", nil)
	if err != nil {
		return err
	}

	if bundle.Policy == nil {
		return fmt.Errorf("bundle for %s contains nil pointer policy", certificateName)
	}
	if bundle.Policy.IssuerParameters == nil {
		return fmt.Errorf("bundle for %s contains nil pointer policy issuer parameters", certificateName)
	}
	if bundle.Policy.IssuerParameters.Name == nil {
		return fmt.Errorf("bundle for %s contains nil pointer policy issuer parameters name", certificateName)
	}

	if *bundle.Policy.IssuerParameters.Name != issuerName {
		policy, err := clusterKeyvault.GetCertificatePolicy(ctx, certificateName, nil)
		if err != nil {
			return err
		}

		policy.IssuerParameters.Name = &issuerName
		_, err = clusterKeyvault.UpdateCertificatePolicy(ctx, certificateName, policy.CertificatePolicy, nil)
		if err != nil {
			return err
		}

		_, err = clusterKeyvault.CreateCertificate(ctx, certificateName, azcertificates.SignedCertificateParameters(issuerName, dnsName, azcertificates.EkuServerAuth), nil)
		if err != nil {
			return err
		}
	}
	return nil
}
