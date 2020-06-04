package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/ready"
)

var _ = Describe("Genevalogging repair", func() {
	Specify("genevalogging must be repaired if deployment deleted", func() {
		mdsdReady := ready.CheckDaemonSetIsReady(Clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging"), "mdsd")

		isReady, err := mdsdReady()
		Expect(err).NotTo(HaveOccurred())
		Expect(isReady).To(Equal(true))

		// delete the daemonset
		err = Clients.Kubernetes.AppsV1().DaemonSets("openshift-azure-logging").Delete("mdsd", nil)
		Expect(err).NotTo(HaveOccurred())

		// wait for it to be fixed
		err = wait.PollImmediate(10*time.Second, 10*time.Minute, mdsdReady)
		Expect(err).NotTo(HaveOccurred())
	})
})
