package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/ready"
)

const (
	machineSetsNamespace = "openshift-machine-api"
)

var _ = Describe("Node check", func() {
	Specify("node count should match the cluster resource and nodes should be ready", func() {
		ctx := context.Background()
		machinesets, err := clients.MachineAPI.MachineV1beta1().MachineSets(machineSetsNamespace).List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		expectedNodeCount := 3 // for masters
		for _, machineset := range machinesets.Items {
			expectedNodeCount += int(*machineset.Spec.Replicas)
		}
		// another hack: we don't currently instantaneously expect all nodes to
		// be ready, it could be that the workaround operator is busy rotating
		// them, which we don't currently wait for on create
		err = wait.PollImmediate(10*time.Second, 30*time.Minute, func() (bool, error) {
			nodes, err := clients.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Warn(err)
				return false, nil // swallow error
			}

			var nodeCount int
			for _, node := range nodes.Items {
				if ready.NodeIsReady(&node) {
					nodeCount++
				} else {
					for _, c := range node.Status.Conditions {
						log.Warnf("node %s status %s", node.Name, c.String())
					}
				}
			}

			return nodeCount == expectedNodeCount, nil
		})
		Expect(err).NotTo(HaveOccurred())
	})
})
