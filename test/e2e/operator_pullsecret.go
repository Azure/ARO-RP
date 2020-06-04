package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

func pullSecretExists(namespace string, name string) (done bool, err error) {
	_, err = Clients.Kubernetes.CoreV1().Secrets(pullSecretName.Namespace).Get(pullSecretName.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

var _ = Describe("Pull secret fix", func() {
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
})
