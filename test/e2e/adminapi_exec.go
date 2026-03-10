//go:build e2e
// +build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("[Admin API] Exec into container action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	const (
		execTestNamespace     = "openshift-azure-operator"
		execTestContainerName = "worker"
	)

	It("must stream command output from a running container", func(ctx context.Context) {
		execTestPodName := fmt.Sprintf("e2e-exec-test-%d-%s", GinkgoParallelProcess(), utilrand.String(5))
		By("creating a long-running test pod in openshift-azure-operator")
		pod := &corev1.Pod{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: execTestPodName, Namespace: execTestNamespace},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:    execTestContainerName,
					Image:   "image-registry.openshift-image-registry.svc:5000/openshift/cli:latest",
					Command: []string{"/bin/bash", "-c", "sleep 300"},
				}},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		}
		CreateK8sObjectWithRetry(
			ctx, clients.Kubernetes.CoreV1().Pods(execTestNamespace).Create, pod, metav1.CreateOptions{},
		)

		defer func() {
			By("deleting the test pod")
			DeleteK8sObjectWithRetry(
				ctx, clients.Kubernetes.CoreV1().Pods(execTestNamespace).Delete, execTestPodName, metav1.DeleteOptions{},
			)
		}()

		By("waiting for the pod to reach Running phase")
		Eventually(func(g Gomega, ctx context.Context) {
			pod = GetK8sObjectWithRetry(
				ctx, clients.Kubernetes.CoreV1().Pods(execTestNamespace).Get, execTestPodName, metav1.GetOptions{},
			)
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
				"pod %s phase: expected Running, got %s", execTestPodName, pod.Status.Phase)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed(),
			"pod %s did not reach Running phase within timeout", execTestPodName)

		By("executing a command via the RP admin exec API")
		reqBody := map[string]string{
			"namespace": execTestNamespace,
			"podName":   execTestPodName,
			"container": execTestContainerName,
			"command":   "echo hello-from-exec",
		}
		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/exec", nil, false, reqBody, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /exec transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from exec endpoint, got %d", resp.StatusCode)

		By("verifying the command output is present in the response")
		Expect(output).To(ContainSubstring("hello-from-exec"),
			"expected 'hello-from-exec' in streamed output:\n%s", output)
		Expect(output).To(ContainSubstring("Done."),
			"expected 'Done.' sentinel in streamed output:\n%s", output)
	})

	It("must return 403 when targeting a customer namespace", func(ctx context.Context) {
		reqBody := map[string]string{
			"namespace": "customer-app",
			"podName":   "some-pod",
			"container": "worker",
			"command":   "echo hello",
		}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/exec", nil, false, reqBody, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /exec transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusForbidden),
			"expected 403 Forbidden for customer namespace, got %d", resp.StatusCode)
	})

	It("must return 400 when request body is missing", func(ctx context.Context) {
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/exec", nil, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /exec transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for missing body, got %d", resp.StatusCode)
	})

	It("must stream stderr alongside stdout", func(ctx context.Context) {
		stderrPodName := fmt.Sprintf("e2e-exec-stderr-%d-%s", GinkgoParallelProcess(), utilrand.String(5))
		By("creating a long-running test pod in openshift-azure-operator")
		pod := &corev1.Pod{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: stderrPodName, Namespace: execTestNamespace},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:    execTestContainerName,
					Image:   "image-registry.openshift-image-registry.svc:5000/openshift/cli:latest",
					Command: []string{"/bin/bash", "-c", "sleep 300"},
				}},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		}
		CreateK8sObjectWithRetry(
			ctx, clients.Kubernetes.CoreV1().Pods(execTestNamespace).Create, pod, metav1.CreateOptions{},
		)
		defer func() {
			DeleteK8sObjectWithRetry(
				ctx, clients.Kubernetes.CoreV1().Pods(execTestNamespace).Delete, stderrPodName, metav1.DeleteOptions{},
			)
		}()

		Eventually(func(g Gomega, ctx context.Context) {
			pod = GetK8sObjectWithRetry(
				ctx, clients.Kubernetes.CoreV1().Pods(execTestNamespace).Get, stderrPodName, metav1.GetOptions{},
			)
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
				"pod %s phase: expected Running, got %s", stderrPodName, pod.Status.Phase)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed(),
			"pod %s did not reach Running phase within timeout", stderrPodName)

		By("executing a command that writes to stderr")
		reqBody := map[string]string{
			"namespace": execTestNamespace,
			"podName":   stderrPodName,
			"container": execTestContainerName,
			"command":   "echo out-line; echo err-line >&2",
		}
		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/exec", nil, false, reqBody, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /exec transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from exec endpoint, got %d", resp.StatusCode)

		By("verifying both stdout and stderr appear in the response")
		Expect(output).To(ContainSubstring("out-line"),
			"expected 'out-line' (stdout) in streamed output:\n%s", output)
		Expect(output).To(SatisfyAny(ContainSubstring("err-line"), ContainSubstring("stderr:")),
			"expected 'err-line' or 'stderr:' (stderr) in streamed output:\n%s", output)
	})
})
