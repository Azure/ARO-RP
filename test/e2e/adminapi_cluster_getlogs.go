package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("[Admin API] Kubernetes get pod logs action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	const containerName = "e2e-test-container-name"
	const podName = "e2e-test-pod-name"

	When("in a standard openshift namespace", func() {
		const namespace = "openshift-azure-operator"

		It("should be able to get logs from a container of a pod", func() {
			ctx := context.Background()
			testGetPodLogsOK(ctx, containerName, podName, namespace)
		})
	})

	When("in a customer namespace", func() {
		const namespace = "e2e-test-namespace"

		It("should be not be able to get logs from customer workload namespaces", func() {
			ctx := context.Background()
			testGetPodLogsFromCustomerNamespaceForbidden(ctx, containerName, podName, namespace)
		})
	})
})

// We will create a pod with known logs of its container and will compare the logs gotten through the kubernetes client and through the Admin API.
func testGetPodLogsOK(ctx context.Context, containerName, podName, namespace string) {
	By("creating a pod in openshift-azure-operator namespace with some known logs then comparing those logs with the logs of the container in the pod created")
	pod := mockPod(containerName, podName, namespace)
	pod, err := clients.Kubernetes.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err = clients.Kubernetes.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	}()
	err = wait.PollInfinite(time.Second*5, func() (done bool, err error) {
		pod, err = clients.Kubernetes.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return true, nil
		case corev1.PodPending:
			if pod.CreationTimestamp.Time.Add(5*time.Minute).Unix() < time.Now().Unix() {
				return false, errors.New("pod was pending for more than 5min")
			}
			return false, nil
		case corev1.PodFailed:
			return true, errors.New(pod.Status.Message)
		}
		return false, nil
	})
	Expect(err).NotTo(HaveOccurred())

	params := url.Values{
		"container": []string{containerName},
		"namespace": []string{namespace},
		"podname":   []string{podName},
	}

	var logs string
	resp, err := adminRequest(ctx, http.MethodGet, "/admin"+resourceIDFromEnv()+"/kubernetespodlogs", params, nil, &logs)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	podOptions := corev1.PodLogOptions{
		Container: pod.Spec.Containers[0].Name,
	}
	r := clients.Kubernetes.CoreV1().Pods(namespace).GetLogs(pod.Name, &podOptions)
	result, err := r.Do(ctx).Raw()
	Expect(err).NotTo(HaveOccurred())
	result = append(result, "\n"...)
	Expect(logs).To(Equal(string(result)))
}

func testGetPodLogsFromCustomerNamespaceForbidden(ctx context.Context, containerName, podName, namespace string) {
	params := url.Values{
		"container": []string{containerName},
		"namespace": []string{namespace},
		"podname":   []string{podName},
	}
	resp, err := adminRequest(ctx, http.MethodGet, "/admin"+resourceIDFromEnv()+"/kubernetespodlogs", params, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func mockPod(containerName, podName, namespace string) *corev1.Pod {
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
				Command: []string{"/bin/bash", "-c", "echo 'mock-pod-logs'"},
			}},
			RestartPolicy: "Never",
			HostNetwork:   true,
			HostPID:       true,
		},
		Status: corev1.PodStatus{},
	}
}
