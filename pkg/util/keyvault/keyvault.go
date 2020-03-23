package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509/pkix"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"

	basekeyvault "github.com/Azure/ARO-RP/pkg/util/azureclient/keyvault"
)

type Eku string

const (
	EkuServerAuth Eku = "1.3.6.1.5.5.7.3.1"
)

type Issuer string

const (
	IssuerDigicert Issuer = "digicert01"
)

type Manager interface {
	basekeyvault.BaseClient

	CreateSignedCertificate(ctx context.Context, keyvaultURI string, issuer Issuer, certificateName, commonName string, eku Eku) error
	EnsureCertificateDeleted(ctx context.Context, keyvaultURI, certificateName string) error
	WaitForCertificateOperation(ctx context.Context, keyvaultURI, certificateName string) error
}

type manager struct {
	basekeyvault.BaseClient
}

func NewManager(kvAuthorizer autorest.Authorizer) Manager {
	return &manager{
		BaseClient: basekeyvault.New(kvAuthorizer),
	}
}

func (m *manager) CreateSignedCertificate(ctx context.Context, keyvaultURI string, issuer Issuer, certificateName, commonName string, eku Eku) error {
	op, err := m.BaseClient.CreateCertificate(ctx, keyvaultURI, certificateName, keyvault.CertificateCreateParameters{
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
					string(eku),
				},
				KeyUsage: &[]keyvault.KeyUsageType{
					keyvault.DigitalSignature,
					keyvault.KeyEncipherment,
				},
				ValidityInMonths: to.Int32Ptr(12),
			},
			IssuerParameters: &keyvault.IssuerParameters{
				Name: to.StringPtr(string(issuer)),
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = checkOperation(&op)
	return err
}

func (m *manager) EnsureCertificateDeleted(ctx context.Context, keyvaultURI, certificateName string) error {
	_, err := m.BaseClient.DeleteCertificate(ctx, keyvaultURI, certificateName)
	if detailedError, ok := err.(autorest.DetailedError); ok {
		if requestError, ok := detailedError.Original.(*azure.RequestError); ok &&
			requestError.ServiceError != nil &&
			requestError.ServiceError.Code == "CertificateNotFound" {
			err = nil
		}
	}

	return err
}

func (m *manager) WaitForCertificateOperation(ctx context.Context, keyvaultURI, certificateName string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		op, err := m.BaseClient.GetCertificateOperation(ctx, keyvaultURI, certificateName)
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
