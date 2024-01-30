package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	DefaultTimeout  = 5 * time.Second
	PollingInterval = 250 * time.Millisecond
)

type allowedK8sOptions interface {
	metav1.CreateOptions | metav1.DeleteOptions | metav1.GetOptions
}

type K8sFunc[T any, U allowedK8sOptions] func(ctx context.Context, name string, options U) (T, error)

// This function takes a k8s function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get
// and the parameters for it and returns the result after asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func PerformK8sCallWithRetry[T any, U allowedK8sOptions](ctx context.Context, k8sFunction K8sFunc[T, U], name string, options U) (result T, err error) {
	Eventually(func(g Gomega, ctx context.Context) {
		result, err = k8sFunction(ctx, name, options)
		g.Expect(err).NotTo(HaveOccurred())
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return
}
