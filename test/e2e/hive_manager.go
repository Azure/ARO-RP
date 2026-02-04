package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api/admin"
)

var _ = Describe("Hive manager creates a namespace", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	var adminAPICluster *admin.OpenShiftCluster

	BeforeEach(func(ctx context.Context) {
		adminAPICluster = adminGetCluster(Default, ctx, clusterResourceID)

		skipIfNotHiveManagedCluster(adminAPICluster)
	})

	var ns *corev1.Namespace

	AfterEach(func() {
		if ns != nil {
			Eventually(func() error {
				return clients.HiveAKS.CoreV1().Namespaces().Delete(context.Background(), ns.Name, metav1.DeleteOptions{})
			}).WithTimeout(20 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
		}
	})

	It("Should be created successfully", func() {
		const docID = "00000000-0000-0000-0000-000000000000"
		var err error
		ns, err = clients.HiveClusterManager.CreateNamespace(context.Background(), docID)
		Expect(err).NotTo(HaveOccurred())
		Expect(ns).NotTo(BeNil())

		res, err := clients.HiveAKS.CoreV1().Namespaces().Get(context.Background(), ns.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())

		Expect(res.String()).To(Equal(ns.String()))
	})
})
