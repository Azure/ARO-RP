package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
)

var (
	DefaultTimeout  = 5 * time.Second
	PollingInterval = 250 * time.Millisecond
)

type K8sGetFunc[T kruntime.Object] func(ctx context.Context, name string, options metav1.GetOptions) (T, error)
type K8sListFunc[T kruntime.Object] func(ctx context.Context, options metav1.ListOptions) (T, error)
type K8sCreateFunc[T kruntime.Object] func(ctx context.Context, object T, options metav1.CreateOptions) (T, error)
type K8sDeleteFunc func(ctx context.Context, name string, options metav1.DeleteOptions) error

// This function takes a get function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get
// and the parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func GetK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, get K8sGetFunc[T], name string, options metav1.GetOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := get(ctx, name, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function takes a list function like clients.Kubernetes.CoreV1().Nodes().List and the
// parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func ListK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, list K8sListFunc[T], options metav1.ListOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := list(ctx, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function takes a create function like clients.Kubernetes.CoreV1().Pods(namespace).Create
// and the parameters for it. It then makes the call with some retry logic and returns the result after
// asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func CreateK8sObjectWithRetry[T kruntime.Object](
	ctx context.Context, create K8sCreateFunc[T], obj T, options metav1.CreateOptions,
) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := create(ctx, obj, options)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function takes a delete function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Delete
// and the parameters for it. It then makes the call with some retry logic.
//
// By default the call is retried for 5s with a 250ms interval.
func DeleteK8sObjectWithRetry(
	ctx context.Context, delete K8sDeleteFunc, name string, options metav1.DeleteOptions,
) {
	Eventually(func(g Gomega, ctx context.Context) {
		err := delete(ctx, name, options)
		g.Expect(err).NotTo(HaveOccurred())
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
}
