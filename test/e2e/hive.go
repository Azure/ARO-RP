package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/hive"
)

var (
	clusterPlatformLabelKey string = "hive.openshift.io/cluster-platform"
	clusterRegionLabelKey   string = "hive.openshift.io/cluster-region"
	clusterPlatformValue    string = "azure"
)

var _ = Describe("Hive-managed ARO cluster", func() {
	var oc *admin.OpenShiftCluster
	var resourceID string

	BeforeEach(func(ctx context.Context) {
		resourceID = resourceIDFromEnv()
		oc = adminGetCluster(Default, ctx, resourceID)

		if oc.Properties.HiveProfile == (admin.HiveProfile{}) {
			Skip("skipping tests because this ARO cluster has not been created/adopted by Hive")
		}
	})

	It("has been properly created/adopted by Hive", func(ctx context.Context) {
		By("verifying that a corresponding ClusterDeployment object exists in the expected namespace in the Hive cluster")
		cd, err := clients.Hive.HiveV1().ClusterDeployments(oc.Properties.HiveProfile.Namespace).Get(ctx, hive.ClusterDeploymentName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("verifying that the ClusterDeployment object has the expected name and labels")
		Expect(cd.ObjectMeta).NotTo(BeNil())
		Expect(cd.ObjectMeta.Name).To(Equal(hive.ClusterDeploymentName))
		Expect(cd.ObjectMeta.Labels).Should(HaveKey(clusterPlatformLabelKey))
		Expect(cd.ObjectMeta.Labels[clusterPlatformLabelKey]).To(Equal(clusterPlatformValue))
		Expect(cd.ObjectMeta.Labels).Should(HaveKey(clusterRegionLabelKey))
		Expect(cd.ObjectMeta.Labels[clusterRegionLabelKey]).To(Equal(oc.Location))

		By("verifying that the ClusterDeployment object spec correctly includes the ARO cluster's Azure region and RG name")
		Expect(cd.Spec).NotTo(BeNil())
		Expect(cd.Spec.ClusterName).To(Equal(oc.Name))
		Expect(cd.Spec.Platform).NotTo(BeNil())
		Expect(cd.Spec.Platform.Azure).NotTo(BeNil())
		Expect(cd.Spec.Platform.Azure.BaseDomainResourceGroupName).To(Equal(oc.Properties.ClusterProfile.ResourceGroupID))
		Expect(cd.Spec.Platform.Azure.Region).To(Equal(oc.Location))
	})
})
