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

type K8sGetFunc[T any] func(ctx context.Context, name string, getOptions metav1.GetOptions) (T, error)
type K8sCreateFunc[T any] func(ctx context.Context, object T, createOptions metav1.CreateOptions) (T, error)
type K8sDeleteFunc func(ctx context.Context, name string, deleteOptions metav1.DeleteOptions) error

// This function takes a get function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get
// and the parameters for it and returns the result after asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func GetK8sObjectWithRetry[T any](ctx context.Context, get K8sGetFunc[T], name string, getOptions metav1.GetOptions) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := get(ctx, name, getOptions)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function takes a create function like clients.Kubernetes.CoreV1().Pods(namespace).Create
// and the parameters for it and returns the result after asserting there were no errors.
//
// By default the call is retried for 5s with a 250ms interval.
func CreateK8sObjectWithRetry[T any](ctx context.Context, create K8sCreateFunc[T], objectToBeCreated T, createOptions metav1.CreateOptions) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := create(ctx, objectToBeCreated, createOptions)
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}

// This function takes a delete function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Delete
// and the parameters for it and returns any possible .
//
// By default the call is retried for 5s with a 250ms interval.
func DeleteK8sObjectWithRetry(ctx context.Context, delete K8sDeleteFunc, name string, deleteOptions metav1.DeleteOptions) {
	Eventually(func(g Gomega, ctx context.Context) {
		err := delete(ctx, name, deleteOptions)
		g.Expect(err).NotTo(HaveOccurred())
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
}

// type allowedK8sOptions interface {
// 	metav1.CreateOptions | metav1.DeleteOptions | metav1.GetOptions
// }

// type K8sFunc[T any, U allowedK8sOptions] func(ctx context.Context, name string, options U) (T, error)

// // This function takes a k8s function like clients.Kubernetes.CertificatesV1().CertificateSigningRequests().Get
// // and the parameters for it and returns the result after asserting there were no errors.
// //
// // By default the call is retried for 5s with a 250ms interval.
// func PerformK8sCallWithRetry[T any, U allowedK8sOptions](ctx context.Context, k8sFunction K8sFunc[T, U], name string, options U) (result T, err error) {
// 	Eventually(func(g Gomega, ctx context.Context) {
// 		result, err = k8sFunction(ctx, name, options)
// 		g.Expect(err).NotTo(HaveOccurred())
// 	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
// 	return
// }
