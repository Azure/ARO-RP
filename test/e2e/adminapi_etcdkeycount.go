//go:build e2e
// +build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// etcdKeyCountLineRx matches a single line of the key count output:
// an integer count, a single space, and a namespace or cluster-scope label.
// Example: "1234 openshift-monitoring"
var etcdKeyCountLineRx = regexp.MustCompile(`^\d+ \S+$`)

var _ = Describe("[Admin API] ETCD key count action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must return key counts from a real etcd pod", func(ctx context.Context) {
		By("finding a master node to target")
		nodes := ListK8sObjectWithRetry(
			ctx,
			clients.Kubernetes.CoreV1().Nodes().List,
			metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/master"},
		)
		Expect(nodes.Items).NotTo(BeEmpty(), "expected at least one master node")
		vmName := nodes.Items[0].Name
		podName := "etcd-" + vmName

		By("verifying the etcd pod is ready before attempting to exec")
		Eventually(func(g Gomega, ctx context.Context) {
			pod := GetK8sObjectWithRetry(
				ctx, clients.Kubernetes.CoreV1().Pods("openshift-etcd").Get, podName, metav1.GetOptions{},
			)
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
				"pod %s phase: expected Running, got %s", podName, pod.Status.Phase)
			// Verify pod is actually ready (not just Running)
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady {
					g.Expect(cond.Status).To(Equal(corev1.ConditionTrue),
						"pod %s Ready condition: expected True, got %s", podName, cond.Status)
					return
				}
			}
			g.Expect(false).To(BeTrue(), "pod %s has no Ready condition", podName)
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed(),
			"etcd pod %s not ready within timeout", podName)

		By("calling the etcdkeycount admin API with vmName=" + vmName)
		params := url.Values{"vmName": []string{vmName}}
		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdkeycount", params, false, nil, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdkeycount transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from etcdkeycount endpoint, got %d", resp.StatusCode)

		By("verifying the key count runs to successful completion")
		Expect(output).To(ContainSubstring("Executing in"),
			"expected 'Executing in' preamble in streamed output:\n%s", output)
		Expect(output).To(ContainSubstring("Done."),
			"expected 'Done.' sentinel in streamed output:\n%s", output)

		By("verifying the response contains non-empty key count output")
		// Output includes the "Executing in …" preamble; filter to data lines only.
		var dataLines []string
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if etcdKeyCountLineRx.MatchString(line) {
				dataLines = append(dataLines, line)
			}
		}
		Expect(dataLines).NotTo(BeEmpty(), "expected at least one '<count> <namespace>' line in response:\n%s", output)
	})
})
