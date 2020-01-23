package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/test/util/ready"
)

var _ = Describe("Check the node count is correct [CheckNodeCount][EveryPR]", func() {
	It("should be possible to list nodes and confirm they are as expected", func() {
		By("Verifying that the expected number of nodes exist and are ready")
		nodes, err := Clients.openshiftclient.CoreV1.Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		var nodeCount int64
		for _, node := range nodes.Items {
			if !strings.HasPrefix(node.Name, "master-") &&
				!strings.HasPrefix(node.Name, "worker-") &&
				ready.NodeIsReady(&node) {
				nodeCount++
			} else {
				for _, c := range node.Status.Conditions {
					Clients.log.Warnf("node %s status %s", node.Name, c.String())
				}
			}
		}
		Expect(nodeCount).To(Equal(6))

	})
})
