package azcertificates

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

type Eku string

const (
	EkuServerAuth Eku = "1.3.6.1.5.5.7.3.1"
	EkuClientAuth Eku = "1.3.6.1.5.5.7.3.2"
)

// SignedCertificateParameters produces a set of parameters used to create a signed certificate
// in KeyVault.
func SignedCertificateParameters(issuer string, commonName string, eku Eku) azcertificates.CreateCertificateParameters {
	return azcertificates.CreateCertificateParameters{
		CertificatePolicy: &azcertificates.CertificatePolicy{
			KeyProperties: &azcertificates.KeyProperties{
				Exportable: to.BoolPtr(true),
				KeyType:    pointerutils.ToPtr(azcertificates.KeyTypeRSA),
				KeySize:    to.Int32Ptr(2048),
			},
			SecretProperties: &azcertificates.SecretProperties{
				ContentType: to.StringPtr("application/x-pem-file"),
			},
			X509CertificateProperties: &azcertificates.X509CertificateProperties{
				Subject: to.StringPtr(pkix.Name{CommonName: getShortCommonName(commonName)}.String()),
				EnhancedKeyUsage: []*string{
					pointerutils.ToPtr(string(eku)),
				},
				SubjectAlternativeNames: &azcertificates.SubjectAlternativeNames{
					DNSNames: []*string{
						pointerutils.ToPtr(commonName),
					},
				},
				KeyUsage: []*azcertificates.KeyUsageType{
					pointerutils.ToPtr(azcertificates.KeyUsageTypeDigitalSignature),
					pointerutils.ToPtr(azcertificates.KeyUsageTypeKeyEncipherment),
				},
				ValidityInMonths: to.Int32Ptr(12),
			},
			LifetimeActions: []*azcertificates.LifetimeAction{
				{
					Trigger: &azcertificates.LifetimeActionTrigger{
						DaysBeforeExpiry: to.Int32Ptr(365 - 90),
					},
					Action: &azcertificates.LifetimeActionType{
						ActionType: pointerutils.ToPtr(azcertificates.CertificatePolicyActionAutoRenew),
					},
				},
			},
			IssuerParameters: &azcertificates.IssuerParameters{
				Name: to.StringPtr(issuer),
			},
		},
	}
}

func getShortCommonName(commonName string) string {
	shortCommonName := commonName
	if len(shortCommonName) > 64 {
		// RFC 5280 requires that the common name be <= 64 characters.  Also see
		// https://docs.digicert.com/manage-certificates/public-certificates-data-entries-that/#64character-maximum-limit-violation .
		// The above does not prevent having longer DNS names in the subject
		// alternative names field.  Key vault does not allow a certificate
		// subject with an empty common name.  So, in the case where the domain
		// name is too long, we use a reserved domain name which cannot be
		// allocated by an end user as the common name.

		// each cloud has different base domain and we have int and prod.
		// the common string is 'aroapp' so we use that to build the base domain for a shorter CN
		baseDomain := shortCommonName[strings.LastIndex(shortCommonName, "aroapp"):]
		shortCommonName = "reserved." + baseDomain
	}
	return shortCommonName
}

func IsCertificateNotFoundError(err error) bool {
	azError := &azcore.ResponseError{}
	if errors.As(err, &azError) {
		return azError.ErrorCode == "CertificateNotFound"
	}
	return false
}

// WaitForCertificateOperation wraps the certificates client to poll for an operation to finish,
// as the Track 2 client still does not support runtime.Poller.
func WaitForCertificateOperation(parent context.Context, log *logrus.Entry, operation func(ctx context.Context) (azcertificates.CertificateOperation, error)) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Minute)
	defer cancel()

	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		op, err := operation(ctx)
		if err != nil {
			return false, err
		}

		return checkOperation(op, log)
	}, ctx.Done())
	return err
}

var errorInfoContains = func(e *azcertificates.ErrorInfo, substr string) bool {
	return e != nil && strings.Contains(e.Error(), substr)
}

func checkOperation(op azcertificates.CertificateOperation, log *logrus.Entry) (bool, error) {
	if op.Status == nil {
		return false, fmt.Errorf("operation status is nil")
	}
	switch *op.Status {
	case "inProgress":
		return false, nil

	case "completed":
		return true, nil

	case "failed":
		// consider failed operation that can retry as not an error, but as if inProgress
		if errorInfoContains(op.Error, "[Status:FailedCanRetry]") {
			if op.StatusDetails != nil {
				log.Warningf("certificateOperation FailedCanRetry %s (%s): Error %v", *op.Status, *op.StatusDetails, op.Error)
			}
			log.Warningf("certificateOperation FailedCanRetry %s: Error %v", *op.Status, op.Error)
			return false, nil
		}
		fallthrough

	default:
		if op.StatusDetails != nil {
			return false, fmt.Errorf("certificateOperation %s (%s): Error %w", *op.Status, *op.StatusDetails, op.Error)
		}
		return false, fmt.Errorf("certificateOperation %s: Error %w", *op.Status, op.Error)
	}
}
