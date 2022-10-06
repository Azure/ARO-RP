package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("List operations", func() {
	Specify("the correct static operations are returned", func() {
		opList, err := clients.Operationsv20200430.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(len(opList) > 0).To(BeTrue())
	})
})
