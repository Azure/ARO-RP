package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Admin API] Must gather action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should return information collected from a cluster cluster", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		By("triggering the mustgather action")
		resp, err := adminRequest(ctx, http.MethodPost, resourceID+"/mustgather", nil, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("ensuring the response is providing the correct filename attachment")
		Expect(resp.Header.Get("content-disposition")).To(Equal(`attachment; filename="must-gather.tgz"`))

		By("ensuring content-type header match expected")
		Expect(resp.Header.Get("content-type")).To(BeElementOf("application/json", "application/gzip"))

		// The body of mustgather is very large. Presently the content/structure is not tested.
	})
})
