package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"go.uber.org/mock/gomock"

	azcertificates_wrapper "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcertificates"
	mock_azcertificates "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcertificates"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
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

			clusterKeyvault := mock_azcertificates.NewMockClient(controller)
			env := mock_env.NewMockInterface(controller)

			if !test.ExpectError {
				clusterKeyvault.EXPECT().GetCertificate(gomock.Any(), test.CertificateName, "", nil).Return(azcertificates.GetCertificateResponse{Certificate: azcertificates.Certificate{
					Policy: &azcertificates.CertificatePolicy{
						IssuerParameters: &azcertificates.IssuerParameters{
							Name: &test.CurrentIssuerName,
						},
					},
				}}, nil)

				if test.CurrentIssuerName != test.NewIssuerName {
					clusterKeyvault.EXPECT().GetCertificatePolicy(gomock.Any(), test.CertificateName, nil).Return(azcertificates.GetCertificatePolicyResponse{CertificatePolicy: azcertificates.CertificatePolicy{
						IssuerParameters: &azcertificates.IssuerParameters{
							Name: &test.CurrentIssuerName,
						},
					}}, nil)

					clusterKeyvault.EXPECT().UpdateCertificatePolicy(gomock.Any(), test.CertificateName, gomock.Any(), nil).Return(azcertificates.UpdateCertificatePolicyResponse{}, nil)
					clusterKeyvault.EXPECT().CreateCertificate(gomock.Any(), test.CertificateName, azcertificates_wrapper.SignedCertificateParameters(test.NewIssuerName, test.DNSName, azcertificates_wrapper.EkuServerAuth), nil).Return(azcertificates.CreateCertificateResponse{}, nil)
				}

				env.EXPECT().ClusterCertificates().AnyTimes().Return(clusterKeyvault)
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
