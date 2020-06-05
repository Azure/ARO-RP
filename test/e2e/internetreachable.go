package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
)

var _ = Describe("Internetreachable shows correct condition", func() {
	Specify("default list should all be reachable", func() {
		co, err := Clients.AROClusters.Clusters().Get("cluster", v1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(co.Status.Conditions.IsTrueFor(aro.InternetReachableFromMaster)).To(BeTrue())
		Expect(co.Status.Conditions.IsTrueFor(aro.InternetReachableFromWorker)).To(BeTrue())
	})
	Specify("custom invalid site shows not reachable", func() {
		var originalSites []string
		// set an unreachable site
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := Clients.AROClusters.Clusters().Get("cluster", v1.GetOptions{})
			if err != nil {
				return err
			}
			originalSites = co.Spec.InternetChecker.Sites
			co.Spec.InternetChecker.Sites = []string{"https://localhost:1234/shouldnotexist"}
			_, err = Clients.AROClusters.Clusters().Update(co)
			return err
		})
		Expect(err).NotTo(HaveOccurred())

		// confirm the conditions are correct
		timeoutCtx, cancel := context.WithTimeout(context.TODO(), 10*time.Minute)
		defer cancel()
		err = wait.PollImmediateUntil(time.Minute, func() (bool, error) {
			co, err := Clients.AROClusters.Clusters().Get("cluster", v1.GetOptions{})
			if err != nil {
				return false, err
			}
			Log.Info(co.Status.Conditions)
			return co.Status.Conditions.IsFalseFor(aro.InternetReachableFromMaster) &&
				co.Status.Conditions.IsFalseFor(aro.InternetReachableFromWorker), nil
		}, timeoutCtx.Done())
		Expect(err).NotTo(HaveOccurred())

		// set the sites back again
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := Clients.AROClusters.Clusters().Get("cluster", v1.GetOptions{})
			if err != nil {
				return err
			}
			co.Spec.InternetChecker.Sites = originalSites
			_, err = Clients.AROClusters.Clusters().Update(co)
			return err
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
