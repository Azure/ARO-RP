package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"

	mgmtredhatopenshift20230401 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2023-04-01/redhatopenshift"
)

var testClusterResourceGroupTags map[string]*string = map[string]*string{
	"e2e_test_tag1": to.StringPtr("foo"),
	"e2e_test_tag2": to.StringPtr("bar"),
}

var _ = Describe("Cluster resource group tags", func() {
	var oc mgmtredhatopenshift20230401.OpenShiftCluster
	var group mgmtfeatures.ResourceGroup
	var err error

	BeforeEach(func(ctx context.Context) {
		oc, group, err = restoreClusterResourceGroupTags(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx context.Context) {
		_, _, err = restoreClusterResourceGroupTags(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	It("must set the tags on the resource group when updated and must never delete tags from the resource group", func(ctx context.Context) {
		By("replacing the current set of tags with a new set")
		updateParameters := mgmtredhatopenshift20230401.OpenShiftClusterUpdate{
			OpenShiftClusterProperties: &mgmtredhatopenshift20230401.OpenShiftClusterProperties{
				ClusterResourceGroupTags: testClusterResourceGroupTags,
			},
		}

		oc, err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, updateParameters)
		Expect(err).NotTo(HaveOccurred())

		By("verifying the request to replace the set of tags fully replaced the previous set stored in the API field")
		Expect(oc.OpenShiftClusterProperties.ClusterResourceGroupTags).To(Equal(testClusterResourceGroupTags))

		By("verifying the new set of tags is present on the resource group")
		group, err = clients.ResourceGroups.Get(ctx, clusterResourceGroupID)
		Expect(err).NotTo(HaveOccurred())

		for k, v := range testClusterResourceGroupTags {
			Expect(group.Tags).Should(HaveKey(k))
			Expect(*group.Tags[k]).To(Equal(*v))
		}

		By("replacing the current set of tags with the empty set")
		updateParameters = mgmtredhatopenshift20230401.OpenShiftClusterUpdate{
			OpenShiftClusterProperties: &mgmtredhatopenshift20230401.OpenShiftClusterProperties{
				ClusterResourceGroupTags: map[string]*string{},
			},
		}

		oc, err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, updateParameters)
		Expect(err).NotTo(HaveOccurred())

		By("verifying that the API field now contains the empty set")
		Expect(oc.OpenShiftClusterProperties.ClusterResourceGroupTags).To(Equal(map[string]*string{}))

		By("verifying that the tags are still present on the resource group despite being removed from the API field")
		group, err = clients.ResourceGroups.Get(ctx, clusterResourceGroupID)
		Expect(err).NotTo(HaveOccurred())

		for k, v := range testClusterResourceGroupTags {
			Expect(group.Tags).Should(HaveKey(k))
			Expect(*group.Tags[k]).To(Equal(*v))
		}
	})
})

func restoreClusterResourceGroupTags(ctx context.Context) (oc mgmtredhatopenshift20230401.OpenShiftCluster, group mgmtfeatures.ResourceGroup, err error) {
	updateParameters := mgmtredhatopenshift20230401.OpenShiftClusterUpdate{
		OpenShiftClusterProperties: &mgmtredhatopenshift20230401.OpenShiftClusterProperties{
			ClusterResourceGroupTags: originalClusterResourceGroupTags,
		},
	}

	oc, err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, updateParameters)
	if err != nil {
		return mgmtredhatopenshift20230401.OpenShiftCluster{}, mgmtfeatures.ResourceGroup{}, err
	}

	group, err = clients.ResourceGroups.Get(ctx, clusterResourceGroupID)
	if err != nil {
		return mgmtredhatopenshift20230401.OpenShiftCluster{}, mgmtfeatures.ResourceGroup{}, err
	}

	group.Tags = originalClusterResourceGroupTagsAzure

	group, err = clients.ResourceGroups.CreateOrUpdate(ctx, clusterResourceGroupID, group)
	if err != nil {
		return mgmtredhatopenshift20230401.OpenShiftCluster{}, mgmtfeatures.ResourceGroup{}, err
	}

	return oc, group, nil
}
