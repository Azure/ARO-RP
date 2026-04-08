package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

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

	It("should not resize when size is already the same", func(ctx context.Context) {
		params := url.Values{
			"deallocateVM": []string{"false"},
			"vmSize":       []string{"Standard_D8s_v5"},
		}

		resp, err := adminRequest(ctx, http.MethodPost,
			"/admin"+clusterResourceID+"/resizecontrolplane",
			params, true, nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})


	It("should do the resize when size is not the same", func(ctx context.Context) {
		params := url.Values{
			"deallocateVM": []string{"false"},
			"vmSize":       []string{"Standard_D8s_v5"},
		}
		var outThing map[string]any
		resp, err := adminRequest(ctx, http.MethodPost,
			"/admin"+clusterResourceID+"/resizecontrolplane",
			params, true, nil, &outThing)
		GinkgoWriter.Printf("Body out: %+v\n", outThing)
		if errors.Is(err, io.EOF) {
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	}, NodeTimeout(30 * time.Minute))
})
