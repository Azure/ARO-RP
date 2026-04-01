//go:build e2e
// +build e2e

package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("[Admin API] ETCD analysis action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must snapshot, run the analysis job, and stream results from a real etcd node", func(ctx context.Context) {
		By("finding a master node to target")
		nodes := ListK8sObjectWithRetry(
			ctx,
			clients.Kubernetes.CoreV1().Nodes().List,
			metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/master"},
		)
		Expect(nodes.Items).NotTo(BeEmpty(), "expected at least one master node")
		vmName := nodes.Items[0].Name

		By("calling the etcdanalysis admin API with vmName=" + vmName)
		params := url.Values{"vmName": []string{vmName}}
		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdanalysis", params, false, nil, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdanalysis transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from etcdanalysis endpoint, got %d", resp.StatusCode)

		By("verifying the snapshot phase completed")
		Expect(output).To(ContainSubstring("Creating etcd snapshot on "+vmName),
			"expected snapshot preamble in streamed output:\n%s", output)
		Expect(output).To(ContainSubstring("Snapshot created."),
			"expected 'Snapshot created.' in streamed output:\n%s", output)

		By("verifying the analysis job ran and cleaned up")
		Expect(output).To(ContainSubstring("Job succeeded."),
			"expected 'Job succeeded.' in streamed output:\n%s", output)
		Expect(output).To(ContainSubstring("Cleanup complete."),
			"expected 'Cleanup complete.' in streamed output:\n%s", output)

		By("verifying no etcd-analysis-privileged-* ServiceAccounts remain in openshift-etcd")
		Eventually(func(g Gomega) {
			sas, err := clients.Kubernetes.CoreV1().ServiceAccounts("openshift-etcd").List(ctx, metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred(), "listing ServiceAccounts in openshift-etcd")
			for _, sa := range sas.Items {
				g.Expect(strings.HasPrefix(sa.Name, "etcd-analysis-privileged-")).To(BeFalse(),
					"ServiceAccount %q should have been deleted by RP cleanup", sa.Name)
			}
		}, "30s", "1s").Should(Succeed(), "RP did not delete etcd-analysis-privileged-* ServiceAccounts within 30s")
	})

	It("must return 400 when vmName is missing", func(ctx context.Context) {
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdanalysis", nil, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdanalysis transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for missing vmName, got %d", resp.StatusCode)
	})

	It("must return 400 when vmName is invalid", func(ctx context.Context) {
		params := url.Values{"vmName": []string{"invalid name!"}}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdanalysis", params, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdanalysis transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for invalid vmName, got %d", resp.StatusCode)
	})
})
