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
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// errNoDefaultCertificate means a cluster has no default cert reference.
	// This can happen because of the following reasons:
	//   1. A cluster doesn't use a managed domain.
	//    	For example it was created with a custom domain)
	//   	or in a dev env where we don't have managed domains.
	//   2. When a customer changed the ingress config incorrectly.
	//
	// While the first is valid the second is something we should be aware of.
	errNoDefaultCertificate = errors.New("ingress has no default certificate set")
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
	cv := &configv1.ClusterVersion{}
	err := r.client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return err
	}

	ingress := &operatorv1.IngressController{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: "openshift-ingress-operator", Name: "default"}, ingress)
	if err != nil {
		return err
	}

	if ingress.Spec.DefaultCertificate == nil {
		return errNoDefaultCertificate
	}

	if ingress.Spec.DefaultCertificate.Name != string(cv.Spec.ClusterID)+"-ingress" {
		return fmt.Errorf("custom ingress certificate in use: %q", ingress.Spec.DefaultCertificate.Name)
	}

	return nil
}
