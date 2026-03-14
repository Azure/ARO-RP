package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"

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
		nodeName := nodes.Items[0].Name

		By("calling the etcdanalysis admin API with nodeName=" + nodeName)
		params := url.Values{"nodeName": []string{nodeName}}
		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdanalysis", params, false, nil, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdanalysis transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from etcdanalysis endpoint, got %d", resp.StatusCode)

		By("verifying the snapshot phase completed")
		Expect(output).To(ContainSubstring("Creating etcd snapshot on "+nodeName),
			"expected snapshot preamble in streamed output:\n%s", output)
		Expect(output).To(ContainSubstring("Snapshot created."),
			"expected 'Snapshot created.' in streamed output:\n%s", output)

		By("verifying the analysis job ran and cleaned up")
		Expect(output).To(ContainSubstring("Job succeeded."),
			"expected 'Job succeeded.' in streamed output:\n%s", output)
		Expect(output).To(ContainSubstring("Cleanup complete."),
			"expected 'Cleanup complete.' in streamed output:\n%s", output)
	})

	It("must return 400 when nodeName is missing", func(ctx context.Context) {
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdanalysis", nil, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdanalysis transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for missing nodeName, got %d", resp.StatusCode)
	})

	It("must return 400 when nodeName is invalid", func(ctx context.Context) {
		params := url.Values{"nodeName": []string{"invalid name!"}}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdanalysis", params, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdanalysis transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for invalid nodeName, got %d", resp.StatusCode)
	})
})
