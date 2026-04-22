package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Admin API] Resize control plane", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should reject an unsupported VM size", func(ctx context.Context) {
		params := url.Values{
			"vmSize":       []string{"Standard_Invalid_Fake"},
			"deallocateVM": []string{"true"},
		}

		resp, err := adminRequest(ctx, http.MethodPost,
			"/admin"+clusterResourceID+"/resizecontrolplane",
			params, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})

	It("should reject a request with missing vmSize", func(ctx context.Context) {
		params := url.Values{
			"deallocateVM": []string{"true"},
		}

		resp, err := adminRequest(ctx, http.MethodPost,
			"/admin"+clusterResourceID+"/resizecontrolplane",
			params, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})
})
