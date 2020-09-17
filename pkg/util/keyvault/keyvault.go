package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
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
	GetSecret(context.Context, string) (keyvault.SecretBundle, error)
	GetSecrets(context.Context) ([]keyvault.SecretItem, error)
	SetSecret(context.Context, string, keyvault.SecretSetParameters) error
	UpgradeCertificatePolicy(context.Context, string) error
	WaitForCertificateOperation(context.Context, string) error
}

type manager struct {
	kv          basekeyvault.BaseClient
	keyvaultURI string
}

// NewManager returns a pointer to a manager containing a BaseClient.  The
// BaseClient is created with kvAuthorizer, which is an authorizer which can
// access a key vault.
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

func (m *manager) GetSecret(ctx context.Context, secretName string) (keyvault.SecretBundle, error) {
	return m.kv.GetSecret(ctx, m.keyvaultURI, secretName, "")
}

func (m *manager) GetSecrets(ctx context.Context) ([]keyvault.SecretItem, error) {
	return m.kv.GetSecrets(ctx, m.keyvaultURI, nil)
}

func (m *manager) SetSecret(ctx context.Context, secretName string, parameters keyvault.SecretSetParameters) error {
	_, err := m.kv.SetSecret(ctx, m.keyvaultURI, secretName, parameters)
	return err
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
