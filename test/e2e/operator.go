package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/util/ready"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

func pullSecretExists(namespace string, name string) (done bool, err error) {
	_, err = Clients.Kubernetes.CoreV1().Secrets(pullSecretName.Namespace).Get(pullSecretName.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

var _ = Describe("ARO Operator", func() {
	Specify("the pull secret should be re-added when deleted", func() {
		// Verify pull secret exists
		_, err := pullSecretExists(pullSecretName.Namespace, pullSecretName.Name)
		Expect(err).NotTo(HaveOccurred())

		// Delete pull secret
		err = Clients.Kubernetes.CoreV1().Secrets(pullSecretName.Namespace).Delete(pullSecretName.Name, &metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Verify operator has re-added\
		err = wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
			return pullSecretExists(pullSecretName.Namespace, pullSecretName.Name)
		})
		Expect(err).NotTo(HaveOccurred())
	})
	Specify("the InternetReachable default list should all be reachable", func() {
		co, err := Clients.AROClusters.Clusters().Get("cluster", v1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(co.Status.Conditions.IsTrueFor(aro.InternetReachableFromMaster)).To(BeTrue())
		Expect(co.Status.Conditions.IsTrueFor(aro.InternetReachableFromWorker)).To(BeTrue())
	})
	Specify("custom invalid site shows not InternetReachable", func() {
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
	Specify("genevalogging must be repaired if deployment deleted", func() {
		mdsdReady := ready.CheckDaemonSetIsReady(Clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging"), "mdsd")

		isReady, err := mdsdReady()
		Expect(err).NotTo(HaveOccurred())
		Expect(isReady).To(Equal(true))

		// delete the daemonset
		err = Clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging").Delete("mdsd", nil)
		Expect(err).NotTo(HaveOccurred())

		// wait for it to be fixed
		err = wait.PollImmediate(10*time.Second, 10*time.Minute, mdsdReady)
		Expect(err).NotTo(HaveOccurred())
	})
})
