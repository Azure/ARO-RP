package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
)

var _ = Describe("[Admin API] Get Hive Cluster Deployment action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	When("A hive managed cluster has its cluster deployment requested", func() {
		It("is managed by hive", func(ctx context.Context) {
			By("requesting the cluster document via RP admin API")
			oc := adminGetCluster(Default, ctx, clusterResourceID)
			By("checking that we received the expected cluster")
			Expect(oc.ID).To(Equal(clusterResourceID))
			By("checking the cluster is managed by hive via its hiveProfile")
			if oc.Properties.HiveProfile.Namespace == "" {
				Skip("Cluster is not managed by hive")
				// NOTE: if we end up with e2e creating clusters via hive we should fail here instead
			}
			clusterDeployment := hivev1.ClusterDeployment{}
			By("requesting the cluster deployment cr")
			resp, err := adminRequest(ctx, http.MethodGet, "/admin"+clusterResourceID+"/clusterdeployment", nil, true, nil, &clusterDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
			Expect(clusterDeployment.Spec.ClusterName).To(Equal(clusterName))
		})
	})
})
