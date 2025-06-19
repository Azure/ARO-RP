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

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	workerMachineLabel = "machine.openshift.io/cluster-api-machine-role=worker"
	workerNodeLabel    = "node-role.kubernetes.io/worker="
)

var _ = Describe("Scale machines", Label(smoke), Serial, Ordered, func() {
	var machineSet *machinev1beta1.MachineSet

	BeforeAll(func(ctx context.Context) {
		By("Fetching machine sets")
		machineSets := ListK8sObjectWithRetry(ctx, clients.MachineAPI.MachineV1beta1().MachineSets("openshift-machine-api").List,
			metav1.ListOptions{})
		Expect(machineSets.Items).To(HaveLen(3))

		By("Scaling up the first machine set")
		machineSet = &machineSets.Items[0]
		machineSet.Spec.Replicas = pointerutils.ToPtr(int32(2))
		machineSet = UpdateK8sObjectWithRetry(ctx, clients.MachineAPI.MachineV1beta1().MachineSets("openshift-machine-api").Update,
			machineSet, metav1.UpdateOptions{})
	})

	It("should be able to scale up and down", func(ctx context.Context) {
		By("Checking there are four machines")
		Eventually(func(g Gomega, ctx context.Context) {
			machines, err := clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").
				List(ctx, metav1.ListOptions{LabelSelector: workerMachineLabel})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(machines.Items).To(HaveLen(4))
			for _, machine := range machines.Items {
				g.Expect(machine).To(Satisfy(isMachineAvailable))
			}
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())

		By("Waiting for all nodes to be available")
		Eventually(func(g Gomega, ctx context.Context) {
			nodes, err := clients.Kubernetes.CoreV1().Nodes().
				List(ctx, metav1.ListOptions{LabelSelector: workerNodeLabel})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(nodes.Items).To(HaveLen(4))
			for _, node := range nodes.Items {
				g.Expect(node).To(Satisfy(isNodeAvailable))
			}
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())

		By("Waiting for a while to ensure the nodes is stable")
		Consistently(func(g Gomega, ctx context.Context) {
			nodes, err := clients.Kubernetes.CoreV1().Nodes().
				List(ctx, metav1.ListOptions{LabelSelector: workerNodeLabel})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(nodes.Items).To(HaveLen(4))
			for _, node := range nodes.Items {
				g.Expect(node).To(Satisfy(isNodeAvailable))
			}
		}, 5*time.Minute, 10*time.Second, ctx).Should(Succeed())
	})

	AfterAll(func(ctx context.Context) {
		By("Scaling down the first machine set")
		machineSet := GetK8sObjectWithRetry(ctx, clients.MachineAPI.MachineV1beta1().MachineSets("openshift-machine-api").Get,
			machineSet.Name, metav1.GetOptions{})
		machineSet.Spec.Replicas = pointerutils.ToPtr(int32(1))
		_ = UpdateK8sObjectWithRetry(ctx, clients.MachineAPI.MachineV1beta1().MachineSets("openshift-machine-api").Update,
			machineSet, metav1.UpdateOptions{})

		By("Checking there are three machines and nodes")
		Eventually(func(g Gomega, ctx context.Context) {
			machines, err := clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").
				List(ctx, metav1.ListOptions{LabelSelector: workerMachineLabel})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(machines.Items).To(HaveLen(3))
			nodes, err := clients.Kubernetes.CoreV1().Nodes().
				List(ctx, metav1.ListOptions{LabelSelector: workerNodeLabel})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(nodes.Items).To(HaveLen(3))
		}, DefaultEventuallyTimeout, 10*time.Second, ctx).Should(Succeed())
	})
})

func isMachineAvailable(machine machinev1beta1.Machine) bool {
	return machine.Status.Phase != nil && *machine.Status.Phase == "Running"
}

func isNodeAvailable(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case corev1.NodeReady:
			if condition.Status != corev1.ConditionTrue {
				return false
			}
		case corev1.NodeMemoryPressure, corev1.NodeDiskPressure, corev1.NodePIDPressure, corev1.NodeNetworkUnavailable:
			if condition.Status != corev1.ConditionFalse {
				return false
			}
		}
	}
	return !node.Spec.Unschedulable
}
