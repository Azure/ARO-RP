package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[Admin API] Kubernetes get pod logs action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	const containerName = "e2e-test-container-name"
	const podName = "e2e-test-pod-name"

	When("in a standard openshift namespace", func() {
		const namespace = "openshift-azure-operator"

		It("must be able to get logs from a container of a pod", func(ctx context.Context) {
			testGetPodLogsOK(ctx, containerName, podName, namespace)
		})
	})

	When("in a customer namespace", func() {
		const namespace = "e2e-test-namespace"

		It("must be not be able to get logs from customer workload namespaces", func(ctx context.Context) {
			testGetPodLogsFromCustomerNamespaceForbidden(ctx, containerName, podName, namespace)
		})
	})
})

// We will create a pod with known logs of its container and will compare the logs gotten through the kubernetes client and through the Admin API.
func testGetPodLogsOK(ctx context.Context, containerName, podName, namespace string) {
	expectedLog := "mock-pod-logs"

	By("creating a test pod in openshift-azure-operator namespace with some known logs")
	pod := mockPod(containerName, podName, namespace, expectedLog)
	pod, err := clients.Kubernetes.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	defer func() {
		By("deleting the test pod")
		err = clients.Kubernetes.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	}()

	By("waiting for the pod to successfully terminate")
	Eventually(func(g Gomega, ctx context.Context) {
		pod, err = clients.Kubernetes.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(pod.Status.Phase).To(Equal(corev1.PodSucceeded))
	}).WithContext(ctx).Should(Succeed())

	By("requesting logs via RP admin API")
	params := url.Values{
		"container": []string{containerName},
		"namespace": []string{namespace},
		"podname":   []string{podName},
	}
	var logs string
	resp, err := adminRequest(ctx, http.MethodGet, "/admin"+clusterResourceID+"/kubernetespodlogs", params, true, nil, &logs)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	By("verifying that logs received from RP match known logs")
	Expect(strings.TrimRight(logs, "\n")).To(Equal(expectedLog))
}

func testGetPodLogsFromCustomerNamespaceForbidden(ctx context.Context, containerName, podName, namespace string) {
	By("requesting logs via RP admin API")
	params := url.Values{
		"container": []string{containerName},
		"namespace": []string{namespace},
		"podname":   []string{podName},
	}

	var logs string
	resp, err := adminRequest(ctx, http.MethodGet, "/admin"+clusterResourceID+"/kubernetespodlogs", params, true, nil, logs)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	By("verifying that we not receive any logs from RP")
	Expect(strings.TrimRight(logs, "\n")).To(BeEmpty())
}

func mockPod(containerName, podName, namespace, fakeLog string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    containerName,
				Image:   "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:103505c93bf45c4a29301f282f1ff046e35b63bceaf4df1cca2e631039289da2",
				Command: []string{"/bin/bash", "-c", fmt.Sprintf("echo %q", fakeLog)},
			}},
			RestartPolicy: "Never",
			HostNetwork:   true,
			HostPID:       true,
		},
		Status: corev1.PodStatus{},
	}
}
