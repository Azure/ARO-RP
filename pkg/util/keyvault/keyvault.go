package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/env"
	basekeyvault "github.com/Azure/ARO-RP/pkg/util/azureclient/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/pem"
)

type Manager interface {
	CreateCertificate(context.Context, string, string) error
	DeleteCertificate(context.Context, string) error
	GetSecret(context.Context, string) (*rsa.PrivateKey, []*x509.Certificate, error)
	WaitForCertificateOperation(context.Context, string) error
}

type manager struct {
	env      env.Interface
	keyvault basekeyvault.BaseClient
}

func NewManager(env env.Interface, localFPKVAuthorizer autorest.Authorizer) Manager {
	return &manager{
		env: env,

		keyvault: basekeyvault.New(localFPKVAuthorizer),
	}
}

func (m *manager) CreateCertificate(ctx context.Context, certificateName, commonName string) error {
	op, err := m.keyvault.CreateCertificate(ctx, m.env.ClustersKeyvaultURI(), certificateName, keyvault.CertificateCreateParameters{
		CertificatePolicy: &keyvault.CertificatePolicy{
			KeyProperties: &keyvault.KeyProperties{
				Exportable: to.BoolPtr(true),
				KeyType:    keyvault.RSA,
				KeySize:    to.Int32Ptr(2048),
			},
			SecretProperties: &keyvault.SecretProperties{
				ContentType: to.StringPtr("application/x-pem-file"),
			},
			X509CertificateProperties: &keyvault.X509CertificateProperties{
				Subject: to.StringPtr(pkix.Name{CommonName: commonName}.String()),
				Ekus: &[]string{
					"1.3.6.1.5.5.7.3.1", // serverAuth
				},
				KeyUsage: &[]keyvault.KeyUsageType{
					keyvault.DigitalSignature,
					keyvault.KeyEncipherment,
				},
				ValidityInMonths: to.Int32Ptr(12),
			},
			IssuerParameters: &keyvault.IssuerParameters{
				Name: to.StringPtr("digicert01"),
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = checkOperation(&op)
	return err
}

func (m *manager) DeleteCertificate(ctx context.Context, certificateName string) error {
	_, err := m.keyvault.DeleteCertificate(ctx, m.env.ClustersKeyvaultURI(), certificateName)
	if detailedError, ok := err.(autorest.DetailedError); ok {
		if requestError, ok := detailedError.Original.(*azure.RequestError); ok &&
			requestError.ServiceError != nil &&
			requestError.ServiceError.Code == "CertificateNotFound" {
			err = nil
		}
	}

	return err
}

func (m *manager) GetSecret(ctx context.Context, secretName string) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	bundle, err := m.keyvault.GetSecret(ctx, m.env.ClustersKeyvaultURI(), secretName, "")
	if err != nil {
		return nil, nil, err
	}

	return pem.Parse([]byte(*bundle.Value))
}

func (m *manager) WaitForCertificateOperation(ctx context.Context, certificateName string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		op, err := m.keyvault.GetCertificateOperation(ctx, m.env.ClustersKeyvaultURI(), certificateName)
		if err != nil {
			return false, err
		}

		return checkOperation(&op)
	}, ctx.Done())
	return err
}

func keyvaultError(err *keyvault.Error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	if err.Code != nil {
		sb.WriteString(*err.Code)
	}

	if err.Message != nil {
		if sb.Len() > 0 {
			sb.WriteString(": ")
		}
		sb.WriteString(*err.Message)
	}

	inner := keyvaultError(err.InnerError)
	if inner != "" {
		if sb.Len() > 0 {
			sb.WriteString(": ")
		}
		sb.WriteString(inner)
	}

	return sb.String()
}
func checkOperation(op *keyvault.CertificateOperation) (bool, error) {
	switch *op.Status {
	case "inProgress":
		return false, nil

	case "completed":
		return true, nil

	default:
		err := keyvaultError(op.Error)
		if op.StatusDetails != nil {
			if err != "" {
				err += ": "
			}
			err += *op.StatusDetails
		}
		return false, fmt.Errorf("certificateOperation %s: %s", *op.Status, err)
	}
}
