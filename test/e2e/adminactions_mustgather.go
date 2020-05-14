package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Admin actions", func() {
	BeforeEach(runAdminTestsInDevOnly)

	Specify("Must gather", func() {
		var headers http.Header
		_, err := adminRequest("POST", "mustgather", "", &headers)
		Expect(err).NotTo(HaveOccurred())

		// Ensure the response is providing the correct filename attachment
		Expect(headers.Get("content-disposition")).To(Equal(`attachment; filename="must-gather.tgz"`))

		// Ensure all of the content-type headers match expected
		for name, val := range headers {
			if name == "content-type" {
				Expect(val).To(BeElementOf("application/json", "application/gzip"))
			}
		}
	})
})
