package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
)

var _ = Describe("Cluster Operators", Label(smoke), func() {
	It("should be all available", func(ctx context.Context) {
		By("checking that all cluster operators are available")
		cos, err := clients.ConfigClient.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		for _, co := range cos.Items {
			available := false
			for _, condition := range co.Status.Conditions {
				if condition.Type == configv1.OperatorAvailable {
					Expect(condition.Status).To(Equal(configv1.ConditionTrue), "operator %s is not available", co.Name)
					available = true
					break
				}
			}
			Expect(available).To(BeTrue(), "operator %s is not available", co.Name)
		}
	})
})

var _ = Describe("ARO Operator", Label(smoke), func() {
	It("should meet all conditions", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			var skipConditions = []string{
				"DefaultIngressCertificate", // This is not enabled in dev clusters and clusters with custom domains.
			}

			for _, condition := range co.Status.Conditions {
				if strings.HasSuffix(condition.Type, "Progressing") || strings.HasSuffix(condition.Type, "Degraded") {
					g.Expect(condition.Status).To(Equal(operatorv1.ConditionFalse))
				} else if slices.Contains(skipConditions, condition.Type) {
					continue
				} else {
					g.Expect(condition.Status).To(Equal(operatorv1.ConditionTrue))
				}
			}
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})
})

var _ = Describe("Node Ready", Label(smoke), func() {
	It("should be ready", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) {
			nodes, err := clients.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			for _, node := range nodes.Items {
				for _, condition := range node.Status.Conditions {
					switch condition.Type {
					case corev1.NodeReady:
						g.Expect(condition.Status).To(Equal(corev1.ConditionTrue))
					case corev1.NodeMemoryPressure, corev1.NodeDiskPressure, corev1.NodePIDPressure, corev1.NodeNetworkUnavailable:
						g.Expect(condition.Status).To(Equal(corev1.ConditionFalse))
					}
				}
				g.Expect(node.Spec.Unschedulable).To(BeFalse())
			}
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
	})
})
