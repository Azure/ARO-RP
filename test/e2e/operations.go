package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List operations", func() {
	Specify("the correct static operations are returned", func() {
		opList, err := Clients.Operations.List(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(len(*opList.Value) > 0).To(BeTrue())
	})
})
