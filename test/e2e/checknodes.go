package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/test/util/ready"
)

var _ = Describe("Check nodes", func() {
	Specify("node count should match the cluster resource and nodes should be ready", func() {
		ctx := context.Background()

		oc, err := Clients.OpenshiftClusters.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("CLUSTER"))
		Expect(err).NotTo(HaveOccurred())

		var expectedNodeCount int32 = 3 // for masters
		for _, wp := range *oc.WorkerProfiles {
			expectedNodeCount += *wp.Count
		}

		nodes, err := Clients.Kubernetes.CoreV1().Nodes().List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		var nodeCount int64
		for _, node := range nodes.Items {
			if ready.NodeIsReady(&node) {
				nodeCount++
			} else {
				for _, c := range node.Status.Conditions {
					Log.Warnf("node %s status %s", node.Name, c.String())
				}
			}
		}

		Expect(nodeCount).To(Equal(expectedNodeCount))
	})
})
