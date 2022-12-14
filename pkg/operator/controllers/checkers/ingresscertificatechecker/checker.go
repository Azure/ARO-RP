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

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	operatorcli operatorclient.Interface
	configcli   configclient.Interface
}

func newIngressCertificateChecker(operatorcli operatorclient.Interface, configcli configclient.Interface) *checker {
	return &checker{
		operatorcli: operatorcli,
		configcli:   configcli,
	}
}

func (r *checker) Check(ctx context.Context) error {
	cv, err := r.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return err
	}

	ingress, err := r.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").Get(ctx, "default", metav1.GetOptions{})
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
