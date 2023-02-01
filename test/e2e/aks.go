//go:build e2e
// +build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tests the kubeconfig ability to get and manipulate the cluster
var _ = Describe("AKS cluster present", Pending, func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	ctx := context.Background()

	// TODO: remove this when all regions have the AKS
	//       since this is going to happen in a weeks,
	//       no need for external configuration option
	//       the list can be found by looking at deployment
	//       pipelines and comparing where AKS was not deployed
	regionsWithoutAKS := []string{
		"australiacentral",
		"australiacentral2",
		"qatarcentral",
		"usgovvirginia",
	}

	It("should get kubeconfig", func() {
		By("region = " + _env.Location())
		for _, region := range regionsWithoutAKS {
			// uses the region information stored in core environment, which reads it from instance metadata.
			if strings.EqualFold(_env.Location(), region) {
				Skip("Region " + region + " does not have AKS, skipping.")
			}
		}

		var err error

		// to avoid name collision by accident
		testNamespaceName := "e2e-test-namespace-" + time.Now().Format("20060102150405")

		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespaceName,
			},
		}

		_, err = clients.HiveAKS.CoreV1().Namespaces().Create(ctx, testNamespace, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		Eventually(func() error {
			_, err := clients.HiveAKS.CoreV1().Namespaces().Get(ctx, testNamespaceName, metav1.GetOptions{})
			return err
		}).WithTimeout(20 * time.Second).WithPolling(1 * time.Second).Should(Succeed())

		Eventually(func() error {
			return clients.HiveAKS.CoreV1().Namespaces().Delete(ctx, testNamespaceName, metav1.DeleteOptions{})
		}).WithTimeout(20 * time.Second).WithPolling(1 * time.Second).Should(Succeed())

	})
})
