// Implements a check that provides detail on potentially faulty or customised
// IngressController configurations on the default controller.
//
// Included checks are:
//  - existence of custom ingress certificate
//  - existence of default ingresscontroller

package ingresscertificatechecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dns"
)

const ingressNameSuffix = "-ingress"

var (
	errNoCertificateAndCustomDomain       = errors.New("missing ingress certificate for cluster with custom domain")
	errNoCertificateAndManagedDomain      = errors.New("missing ingress certificate for cluster with managed domain")
	errInvalidCertificateAndManagedDomain = errors.New("invalid ingress certificate name for cluster with managed domain")
)

type ingressCertificateChecker interface {
	Check(ctx context.Context) error
}

type checker struct {
	client client.Client
}

func newIngressCertificateChecker(client client.Client) *checker {
	return &checker{
		client: client,
	}
}

func (r *checker) Check(ctx context.Context) error {
	cv, err := r.clusterVersion(ctx)
	if err != nil {
		return err
	}

	ingress, err := r.ingress(ctx)
	if err != nil {
		return err
	}

	clusterHasManagedDomain, err := r.clusterHasManagedDomain(ctx)
	if err != nil {
		return err
	}
	return validateCertificate(cv.Spec.ClusterID, ingress.Spec.DefaultCertificate, clusterHasManagedDomain)
}

func (r *checker) ingress(ctx context.Context) (*operatorv1.IngressController, error) {
	ingress := &operatorv1.IngressController{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: "openshift-ingress-operator", Name: "default"}, ingress)
	if err != nil {
		return nil, err
	}
	return ingress, nil
}

func (r *checker) clusterVersion(ctx context.Context) (*configv1.ClusterVersion, error) {
	cv := &configv1.ClusterVersion{}
	err := r.client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return nil, err
	}
	return cv, nil
}

func (r *checker) clusterHasManagedDomain(ctx context.Context) (bool, error) {
	cluster := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
	if err != nil {
		return false, err
	}

	return dns.IsManagedDomain(cluster.Spec.Domain), nil
}

func validateCertificate(clusterId configv1.ClusterID, ingressCertificate *corev1.LocalObjectReference, clusterHasManagedDomain bool) error {
	if ingressCertificate == nil {
		if clusterHasManagedDomain {
			return errNoCertificateAndManagedDomain
		}
		return errNoCertificateAndCustomDomain
	}

	certificateName := ingressCertificate.Name

	if !certificateIsValid(certificateName, clusterId) && clusterHasManagedDomain {
		return errInvalidCertificateAndManagedDomain
	}

	return nil
}

func certificateIsValid(certificateName string, clusterID configv1.ClusterID) bool {
	expectedCertName := string(clusterID) + ingressNameSuffix
	return certificateName == expectedCertName
}
