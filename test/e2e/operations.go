package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List operations", func() {
	Specify("the correct static operations are returned", func() {
		if os.Getenv("RP_MODE") != "development" {
			Skip("Operations api is unreachable calling through ARM. Skipping test.")
		}
		opList, err := Clients.Operations.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(len(opList) > 0).To(BeTrue())
	})
})
