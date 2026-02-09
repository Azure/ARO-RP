package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (k *kubeActions) ApproveCsr(ctx context.Context, csrName string) error {
	csr, err := k.kubecli.CertificatesV1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("certificate signing request '%s' was not found.", csrName))
		}
		return err
	}

	return k.updateCsr(ctx, csr)
}

func (k *kubeActions) ApproveAllCsrs(ctx context.Context) error {
	csrs, err := k.kubecli.CertificatesV1().CertificateSigningRequests().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, csr := range csrs.Items {
		err = k.updateCsr(ctx, &csr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *kubeActions) updateCsr(ctx context.Context, csr *certificatesv1.CertificateSigningRequest) error {
	modifiedCSR, hasCondition, err := addConditionIfNeeded(csr, string(certificatesv1.CertificateDenied), string(certificatesv1.CertificateApproved), "AROSupportApprove", "This CSR was approved by ARO support personnel.")
	if err != nil {
		return err
	}
	if !hasCondition {
		_, err = k.kubecli.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, modifiedCSR.Name, modifiedCSR, metav1.UpdateOptions{})
	}
	return err
}

func addConditionIfNeeded(csr *certificatesv1.CertificateSigningRequest, mustNotHaveConditionType, conditionType, reason, message string) (*certificatesv1.CertificateSigningRequest, bool, error) {
	var alreadyHasCondition bool
	for _, c := range csr.Status.Conditions {
		if string(c.Type) == mustNotHaveConditionType {
			return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "", fmt.Sprintf("certificate signing request %q is already %s", csr.Name, c.Type))
		}
		if string(c.Type) == conditionType {
			alreadyHasCondition = true
		}
	}
	if alreadyHasCondition {
		return csr, true, nil
	}
	csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
		Type:           certificatesv1.RequestConditionType(conditionType),
		Status:         corev1.ConditionTrue,
		Reason:         reason,
		Message:        message,
		LastUpdateTime: metav1.Now(),
	})
	return csr, false, nil
}
