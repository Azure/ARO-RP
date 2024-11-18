package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"go.uber.org/mock/gomock"

	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_keyvault "github.com/Azure/ARO-RP/pkg/util/mocks/keyvault"
)

func TestEnsureCertificateIssuer(t *testing.T) {
	tests := []struct {
		Name              string
		CertificateName   string
		DNSName           string
		CurrentIssuerName string
		NewIssuerName     string
		ExpectError       bool
	}{
		{
			Name:              "current issuer matches new issuer",
			CertificateName:   "testCert",
			DNSName:           "*.apps.test.asdf.tld",
			CurrentIssuerName: "fakeIssuer",
			NewIssuerName:     "fakeIssuer",
		},
		{
			Name:              "current issuer different from new issuer",
			CertificateName:   "testCert",
			DNSName:           "*.apps.test.asdf.tld",
			CurrentIssuerName: "OldFakeIssuer",
			NewIssuerName:     "NewFakeIssuer",
		},
		{
			Name:              "malformed dns data",
			CertificateName:   "testCert",
			DNSName:           "f898896d-c9ce-4d2b-b218-95a1f858df3a-something",
			CurrentIssuerName: "OldFakeIssuer",
			NewIssuerName:     "NewFakeIssuer",
			ExpectError:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			clusterKeyvault := mock_keyvault.NewMockManager(controller)
			env := mock_env.NewMockInterface(controller)

			if !test.ExpectError {
				clusterKeyvault.EXPECT().GetCertificate(gomock.Any(), test.CertificateName).Return(azkeyvault.CertificateBundle{
					Policy: &azkeyvault.CertificatePolicy{
						IssuerParameters: &azkeyvault.IssuerParameters{
							Name: &test.CurrentIssuerName,
						},
					},
				}, nil)

				if test.CurrentIssuerName != test.NewIssuerName {
					clusterKeyvault.EXPECT().GetCertificatePolicy(gomock.Any(), test.CertificateName).Return(azkeyvault.CertificatePolicy{
						IssuerParameters: &azkeyvault.IssuerParameters{
							Name: &test.CurrentIssuerName,
						},
					}, nil)

					clusterKeyvault.EXPECT().UpdateCertificatePolicy(gomock.Any(), test.CertificateName, gomock.Any()).Return(nil)
					clusterKeyvault.EXPECT().CreateSignedCertificate(gomock.Any(), test.NewIssuerName, test.CertificateName, test.DNSName, gomock.Any()).Return(nil)
				}

				env.EXPECT().ClusterKeyvault().AnyTimes().Return(clusterKeyvault)
			}

			m := &manager{
				env: env,
			}

			err := m.ensureCertificateIssuer(context.TODO(), test.CertificateName, test.DNSName, test.NewIssuerName)
			if test.ExpectError == (err == nil) {
				t.Errorf("TestCorrectCertificateIssuer() %s: ExpectError: %t, actual error: %s\n", test.Name, test.ExpectError, err)
			}
		})
	}
}
