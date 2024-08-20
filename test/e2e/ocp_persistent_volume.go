package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var _ = Describe("Persistent Volume", Label(smoke), func() {
	DescribeTable("should provision PVCs", func(ctx context.Context, pvcName string) {
		const namespace = "default"
		manifest := fmt.Sprintf("static_resources/pvc-%s.yaml", pvcName)
		podName := fmt.Sprintf("bb-%s", pvcName)
		By(fmt.Sprintf("Creating a pod with %s", pvcName))

		f, err := staticResources.Open(manifest)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			err = f.Close()
			Expect(err).NotTo(HaveOccurred())
		}()
		pod, err := loadResourcesFromYaml(f)
		Expect(err).NotTo(HaveOccurred())
		createResources(ctx, pod...)

		DeferCleanup(func(ctx context.Context) {
			cleanupResources(ctx, pod...)
		})

		expectPodRunning(ctx, namespace, podName)
		expectPVCBound(ctx, namespace, pvcName)

		pvc := GetK8sObjectWithRetry(ctx, clients.Kubernetes.CoreV1().PersistentVolumeClaims(namespace).Get, pvcName, metav1.GetOptions{})
		pvName := pvc.Spec.VolumeName
		Expect(pvName).NotTo(BeEmpty())
		expectPVBound(ctx, pvName)
	},
		Entry(nil, "azurefile-csi"),
		Entry(nil, "managed-csi"),
		Entry(nil, "managed-csi-encrypted-cmk"),
	)
})

func expectPVCBound(ctx context.Context, namespace, name string) {
	GinkgoHelper()
	By("Checking the PVC status")
	Eventually(func(g Gomega, ctx context.Context) {
		pvc, err := clients.Kubernetes.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pvc.Status.Phase).To(Equal(corev1.ClaimBound))
	}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())
}

func expectPVBound(ctx context.Context, name string) {
	GinkgoHelper()
	By("Checking the PV status")
	Eventually(func(g Gomega, ctx context.Context) {
		pv, err := clients.Kubernetes.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(pv.Status.Phase).To(Equal(corev1.VolumeBound))
	}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())
}
