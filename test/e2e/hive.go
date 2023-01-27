package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/hive"
)

var (
	clusterPlatformLabelKey string = "hive.openshift.io/cluster-platform"
	clusterRegionLabelKey   string = "hive.openshift.io/cluster-region"

	controlPlaneAPIURLOverride = func(clusterDomain string, clusterLocation string) string {
		if !strings.ContainsRune(clusterDomain, '.') {
			clusterDomain += "." + clusterLocation + "." + os.Getenv("PARENT_DOMAIN_NAME")
		}

		return fmt.Sprintf("api-int.%s:6443", clusterDomain)
	}
)

var _ = Describe("Hive-managed ARO cluster", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	var adminAPICluster *admin.OpenShiftCluster

	BeforeEach(func(ctx context.Context) {
		adminAPICluster = adminGetCluster(Default, ctx, clusterResourceID)

		skipIfNotHiveManagedCluster(adminAPICluster)
	})

	It("has been properly created/adopted by Hive", func(ctx context.Context) {
		By("verifying that a corresponding ClusterDeployment object exists in the expected namespace in the Hive cluster")
		cd, err := clients.Hive.HiveV1().ClusterDeployments(adminAPICluster.Properties.HiveProfile.Namespace).Get(ctx, hive.ClusterDeploymentName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("verifying that the ClusterDeployment object has the expected name and labels")
		Expect(cd.ObjectMeta).NotTo(BeNil())
		Expect(cd.ObjectMeta.Name).To(Equal(hive.ClusterDeploymentName))
		Expect(cd.ObjectMeta.Labels).Should(HaveKey(clusterPlatformLabelKey))
		Expect(cd.ObjectMeta.Labels[clusterPlatformLabelKey]).To(Equal("azure"))
		Expect(cd.ObjectMeta.Labels).Should(HaveKey(clusterRegionLabelKey))
		Expect(cd.ObjectMeta.Labels[clusterRegionLabelKey]).To(Equal(adminAPICluster.Location))

		By("verifying that the ClusterDeployment object spec correctly includes the ARO cluster's Azure region and RG name")
		Expect(cd.Spec).NotTo(BeNil())
		Expect(cd.Spec.ClusterName).To(Equal(adminAPICluster.Name))
		Expect(cd.Spec.Platform).NotTo(BeNil())
		Expect(cd.Spec.Platform.Azure).NotTo(BeNil())
		Expect(cd.Spec.Platform.Azure.BaseDomainResourceGroupName).To(Equal(adminAPICluster.Properties.ClusterProfile.ResourceGroupID))
		Expect(cd.Spec.Platform.Azure.Region).To(Equal(adminAPICluster.Location))

		By("verifying that the ClusterDeployment object spec includes the expected ControlPlaneConfig overrides")
		Expect(cd.Spec.ControlPlaneConfig).NotTo(BeNil())
		Expect(cd.Spec.ControlPlaneConfig.APIServerIPOverride).To(Equal(adminAPICluster.Properties.NetworkProfile.APIServerPrivateEndpointIP))
		Expect(cd.Spec.ControlPlaneConfig.APIURLOverride).To(Equal(controlPlaneAPIURLOverride(adminAPICluster.Properties.ClusterProfile.Domain, adminAPICluster.Location)))
	})
})
