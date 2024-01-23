package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	DefaultTimeout  = 5 * time.Second
	PollingInterval = 250 * time.Millisecond
)

type K8sGetFunc[T any] func(ctx context.Context, name string, getOptions metav1.GetOptions) (T, error)

// This function takes a get function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get
// and the parameters for it and returns the result after asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func GetK8sObjectWithRetry[T any](ctx context.Context, get K8sGetFunc[T], name string, getOptions metav1.GetOptions) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := get(ctx, name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}
