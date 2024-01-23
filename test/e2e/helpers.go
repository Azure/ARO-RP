package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// GetObject takes a get function like clientset.CoreV1().Pods(ns).Get
// and the parameters for it and returns a function that executes that get
// operation in a [gomega.Eventually] or [gomega.Consistently].
//
// Delays and retries are handled by [HandleRetry]. A "not found" error is
// a fatal error that causes polling to stop immediately. If that is not
// desired, then wrap the result with [IgnoreNotFound].

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

func GetK8sObjectWithRetry[T any](ctx context.Context, get K8sGetFunc[T], name string, getOptions metav1.GetOptions) T {
	var object T
	Eventually(func(g Gomega, ctx context.Context) {
		result, err := get(ctx, name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		object = result
	}).WithContext(ctx).WithTimeout(DefaultTimeout).WithPolling(PollingInterval).Should(Succeed())
	return object
}
