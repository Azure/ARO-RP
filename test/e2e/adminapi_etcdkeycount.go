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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// etcdKeyCountLineRx matches a single line of the key count output:
// an integer count, a single space, and a required namespace segment.
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

		By("calling the etcdkeycount admin API with vmName=" + vmName)
		params := url.Values{"vmName": []string{vmName}}
		var output string
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdkeycount", params, false, nil, &output)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdkeycount transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusOK),
			"expected 200 OK from etcdkeycount endpoint, got %d", resp.StatusCode)

		By("verifying the response contains non-empty key count output")
		Expect(output).To(ContainSubstring("Executing in"),
			"expected 'Executing in' preamble in streamed output:\n%s", output)
		Expect(output).To(ContainSubstring("Done."),
			"expected 'Done.' sentinel in streamed output:\n%s", output)

		By("verifying at least one output line matches the expected numeric format")
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

	It("must return 400 when vmName is missing", func(ctx context.Context) {
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdkeycount", nil, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdkeycount transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for missing vmName, got %d", resp.StatusCode)
	})

	It("must return 400 when vmName is invalid", func(ctx context.Context) {
		params := url.Values{"vmName": []string{"invalid name!"}}
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/etcdkeycount", params, false, nil, nil)
		Expect(err).NotTo(HaveOccurred(), "POST /etcdkeycount transport error")
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
			"expected 400 Bad Request for invalid vmName, got %d", resp.StatusCode)
	})
})
