package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("List operations", func() {
	It("must return the correct static operations", func(ctx context.Context) {
		opList, err := clients.Operations.List(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(opList) > 0).To(BeTrue())
	})
})
