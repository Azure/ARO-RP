package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"

	basekeyvault "github.com/Azure/ARO-RP/pkg/util/azureclient/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/pem"
)

type Eku string

const (
	EkuServerAuth Eku = "1.3.6.1.5.5.7.3.1"
	EkuClientAuth Eku = "1.3.6.1.5.5.7.3.2"
)

type Issuer string

const (
	IssuerDigicert Issuer = "digicert01"
)

type Manager interface {
	CreateSignedCertificate(context.Context, Issuer, string, string, Eku) error
	EnsureCertificateDeleted(context.Context, string) error
	GetCertificateSecret(context.Context, string) (*rsa.PrivateKey, []*x509.Certificate, error)
	GetSecret(context.Context, string, string) (keyvault.SecretBundle, error)
	GetSecrets(context.Context, *int32) ([]keyvault.SecretItem, error)
	SetSecret(context.Context, string, keyvault.SecretSetParameters) (keyvault.SecretBundle, error)
	UpgradeCertificatePolicy(context.Context, string) error
	WaitForCertificateOperation(context.Context, string) error
}

type manager struct {
	kv basekeyvault.BaseClient

	keyvaultURI string
}

func NewManager(kvAuthorizer autorest.Authorizer, keyvaultURI string) Manager {
	return &manager{
		kv:          basekeyvault.New(kvAuthorizer),
		keyvaultURI: keyvaultURI,
	}
}

func (m *manager) CreateSignedCertificate(ctx context.Context, issuer Issuer, certificateName, commonName string, eku Eku) error {
	op, err := m.kv.CreateCertificate(ctx, m.keyvaultURI, certificateName, keyvault.CertificateCreateParameters{
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
			LifetimeActions: &[]keyvault.LifetimeAction{
				{
					Trigger: &keyvault.Trigger{
						DaysBeforeExpiry: to.Int32Ptr(365 - 90),
					},
					Action: &keyvault.Action{
						ActionType: keyvault.AutoRenew,
					},
				},
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

func (m *manager) EnsureCertificateDeleted(ctx context.Context, certificateName string) error {
	_, err := m.kv.DeleteCertificate(ctx, m.keyvaultURI, certificateName)
	if detailedError, ok := err.(autorest.DetailedError); ok {
		if requestError, ok := detailedError.Original.(*azure.RequestError); ok &&
			requestError.ServiceError != nil &&
			requestError.ServiceError.Code == "CertificateNotFound" {
			err = nil
		}
	}

	return err
}

func (m *manager) GetCertificateSecret(ctx context.Context, secretName string) (*rsa.PrivateKey, []*x509.Certificate, error) {
	bundle, err := m.kv.GetSecret(ctx, m.keyvaultURI, secretName, "")
	if err != nil {
		return nil, nil, err
	}

	key, certs, err := pem.Parse([]byte(*bundle.Value))
	if err != nil {
		return nil, nil, err
	}

	if key == nil {
		return nil, nil, fmt.Errorf("no private key found")
	}

	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificate found")
	}

	return key, certs, nil
}

func (m *manager) GetSecret(ctx context.Context, secretName string, secretVersion string) (keyvault.SecretBundle, error) {
	return m.kv.GetSecret(ctx, m.keyvaultURI, secretName, secretVersion)
}

func (m *manager) GetSecrets(ctx context.Context, maxresults *int32) ([]keyvault.SecretItem, error) {
	return m.kv.GetSecrets(ctx, m.keyvaultURI, maxresults)
}

func (m *manager) SetSecret(ctx context.Context, secretName string, parameters keyvault.SecretSetParameters) (keyvault.SecretBundle, error) {
	return m.kv.SetSecret(ctx, m.keyvaultURI, secretName, parameters)
}

func (m *manager) WaitForCertificateOperation(ctx context.Context, certificateName string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		op, err := m.kv.GetCertificateOperation(ctx, m.keyvaultURI, certificateName)
		if err != nil {
			return false, err
		}

		return checkOperation(&op)
	}, ctx.Done())
	return err
}

func (m *manager) UpgradeCertificatePolicy(ctx context.Context, certificateName string) error {
	policy, err := m.kv.GetCertificatePolicy(ctx, m.keyvaultURI, certificateName)
	if err != nil {
		return err
	}

	lifetimeActions := &[]keyvault.LifetimeAction{
		{
			Trigger: &keyvault.Trigger{
				DaysBeforeExpiry: to.Int32Ptr(365 - 90),
			},
			Action: &keyvault.Action{
				ActionType: keyvault.AutoRenew,
			},
		},
	}

	if reflect.DeepEqual(policy.LifetimeActions, lifetimeActions) {
		return nil
	}

	policy.LifetimeActions = lifetimeActions

	_, err = m.kv.UpdateCertificatePolicy(ctx, m.keyvaultURI, certificateName, policy)
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
