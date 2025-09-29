package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	masterMachineLabel = "machine.openshift.io/cluster-api-machine-role=master"
)

// Steps performed in this test
// 1. Disabling cluster-version-operator and etcd-operator
// 2. Check if there are guardrails preventing machines to be deleted, and disable them if necessary
// 3. Delete first master machine
// 4. Enable operators
// 5. Recreate Machine
// 6. Wait for new ETCD pod
// 7. Run the fix
// 8. Wait until operators recover from degraded
// 9. Enable back guardrails if necessary

var _ = Describe("Master replacement", Label(regressiontest), func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should fix etcd automatically", Serial, func(ctx context.Context) {
		By("Disabling reconciliation")
		dep, err := clients.Kubernetes.AppsV1().Deployments("openshift-cluster-version").Get(ctx, "cluster-version-operator", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		dep.Spec.Replicas = pointerutils.ToPtr(int32(0))
		_, err = clients.Kubernetes.AppsV1().Deployments("openshift-cluster-version").Update(ctx, dep, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		dep, err = clients.Kubernetes.AppsV1().Deployments("openshift-etcd-operator").Get(ctx, "etcd-operator", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		dep.Spec.Replicas = pointerutils.ToPtr(int32(0))
		_, err = clients.Kubernetes.AppsV1().Deployments("openshift-etcd-operator").Update(ctx, dep, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Check if we have guardrails in the cluster, so we can disable them to delete machines if necessary
		templateAroConstraint := &unstructured.Unstructured{}
		templateAroConstraint.SetAPIVersion("constraints.gatekeeper.sh/v1beta1")
		templateAroConstraint.SetKind("ARODenyLabels")
		constraintPresent := true

		aroConstraintClient, err := clients.Dynamic.GetClient(templateAroConstraint)
		Expect(err).NotTo(HaveOccurred())
		_, err = aroConstraintClient.Get(ctx, "aro-machines-deny", metav1.GetOptions{})

		if err != nil {
			if kerrors.IsNotFound(err) {
				// This cluster does not have guardrails, so we don't need to disable and enable again
				constraintPresent = false
			} else {
				// something else happened and we can't continue testing
				Expect(err).ToNot(HaveOccurred())
			}
		}

		if constraintPresent {
			patchPayload := `[
			{
				"op": "replace",
				"path": "/spec/operatorflags/aro.guardrails.policies.aro-machines-deny.managed",
				"value": "false"
			}
		]`
			patchBytes := []byte(patchPayload)
			By("Disabling guardrail policies for aro machines")
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Patch(ctx, "cluster", types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for constraint to be removed")
			Eventually(func(g Gomega, ctx context.Context) {
				_, err := aroConstraintClient.Get(ctx, "aro-machines-deny", metav1.GetOptions{})
				g.Expect(err).To(HaveOccurred())
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}, 10*time.Minute, 10*time.Second, ctx).Should(Succeed())
		}

		By("Deleting the first master machine")
		machines, err := clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").
			List(ctx, metav1.ListOptions{LabelSelector: masterMachineLabel})
		Expect(err).NotTo(HaveOccurred())
		Expect(machines.Items).To(HaveLen(3))
		machine := machines.Items[0]

		machine.Spec.ProviderID = nil
		machine.Status = v1beta1.MachineStatus{}
		machine.Spec.LifecycleHooks = v1beta1.LifecycleHooks{}
		_, err = clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").Update(ctx, &machine, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").Delete(ctx, machine.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the machine to be deleted")
		Eventually(func(g Gomega, ctx context.Context) {
			machines, err := clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").
				List(ctx, metav1.ListOptions{LabelSelector: masterMachineLabel})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(machines.Items).To(HaveLen(2))
		}, 10*time.Minute, 10*time.Second, ctx).Should(Succeed())

		By("Reverting deployments") // cluster-version-operator reconciles etcd-operator.
		dep, err = clients.Kubernetes.AppsV1().Deployments("openshift-cluster-version").Get(ctx, "cluster-version-operator", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		dep.Spec.Replicas = pointerutils.ToPtr(int32(1))
		_, err = clients.Kubernetes.AppsV1().Deployments("openshift-cluster-version").Update(ctx, dep, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Recreating the machine")
		machine.ObjectMeta = metav1.ObjectMeta{
			Labels:    machine.Labels,
			Name:      machine.Name,
			Namespace: machine.Namespace,
		}
		_, err = clients.MachineAPI.MachineV1beta1().Machines("openshift-machine-api").Create(ctx, &machine, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the machine to be created and its node to be ready")
		Eventually(func(g Gomega, ctx context.Context) {
			node, err := clients.Kubernetes.CoreV1().Nodes().Get(ctx, machine.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady {
					g.Expect(condition.Status).To(Equal(corev1.ConditionTrue))
					return
				}
			}
		}, 15*time.Minute, 10*time.Second, ctx).Should(Succeed())

		By("Waiting for the etcd pod to be created")
		Eventually(func(g Gomega, ctx context.Context) {
			_, err := clients.Kubernetes.CoreV1().Pods("openshift-etcd").Get(ctx, fmt.Sprintf("etcd-%s", machine.Name), metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
		}, 5*time.Minute, 10*time.Second, ctx).Should(Succeed())

		By("Running etcd recovery API")
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdrecovery", nil, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		// The master replacement doesn't always break the etcd.
		// It returns 200 and fixes it if broken, and it returns 400 if not broken.
		// If it gets either of them, we can say that the etcd is fixed (or not broken).
		Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest)))
		By(fmt.Sprintf("Status Code: %d", resp.StatusCode))

		By("Waiting for the cluster operator not to be degraded")
		Eventually(func(g Gomega, ctx context.Context) {
			cos, err := clients.ConfigClient.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			for _, co := range cos.Items {
				isDegraded := false
				isAvailable := false
				for _, condition := range co.Status.Conditions {
					if condition.Type == configv1.OperatorAvailable && condition.Status == configv1.ConditionTrue {
						isAvailable = true
					}
					if condition.Type == configv1.OperatorDegraded && condition.Status == configv1.ConditionTrue {
						isDegraded = true
					}
				}
				g.Expect(isAvailable).To(BeTrue(), "operator %s is not available", co.Name)
				g.Expect(isDegraded).To(BeFalse(), "operator %s is degraded", co.Name)
			}
		}, 10*time.Minute, 10*time.Second, ctx).Should(Succeed())

		if constraintPresent {
			// Re-enabling the cluster-api-machine-role label so gatekeeper allows to delete the machine
			By("Enabling guardrail policies for aro machines")
			patchPayload := `[
			{
				"op": "replace",
				"path": "/spec/operatorflags/aro.guardrails.policies.aro-machines-deny.managed",
				"value": "true"
			}
		]`
			patchBytes := []byte(patchPayload)

			_, err = clients.AROClusters.AroV1alpha1().Clusters().Patch(ctx, "cluster", types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	})
})
