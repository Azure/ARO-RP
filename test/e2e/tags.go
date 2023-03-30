package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/to"

	mgmtredhatopenshift20230401 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2023-04-01/redhatopenshift"
)

const nsgName = "e2e-tag-test-nsg"

var testResourceTags map[string]*string = map[string]*string{
	"e2e_test_tag1": to.StringPtr("foo"),
	"e2e_test_tag2": to.StringPtr("bar"),
}

// From experimentation, this is the amount of time needed
// for the tagging policy assignment to work as expected when
// creating and updating resources within its scope.
var policyAssignmentLagDuration = 2 * time.Minute

var _ = Describe("Cluster resource tags", func() {
	var oc mgmtredhatopenshift20230401.OpenShiftCluster
	var group mgmtfeatures.ResourceGroup
	var err error

	BeforeEach(func(ctx context.Context) {
		oc, group, err = restoreClusterResourceTags(ctx)
		Expect(err).NotTo(HaveOccurred())

		err = clients.NetworkSecurityGroups.DeleteAndWait(ctx, clusterResourceGroupID, nsgName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func(ctx context.Context) {
		_, _, err = restoreClusterResourceTags(ctx)
		Expect(err).NotTo(HaveOccurred())

		err = clients.NetworkSecurityGroups.DeleteAndWait(ctx, clusterResourceGroupID, nsgName)
		Expect(err).NotTo(HaveOccurred())
	})

	It("must set the tags on the resource group and resources when updated, must only add/update and never delete tags, and must add the tags to new resources created within the cluster RG", func(ctx context.Context) {
		By("replacing the current set of tags with a new set")
		updateParameters := mgmtredhatopenshift20230401.OpenShiftClusterUpdate{
			OpenShiftClusterProperties: &mgmtredhatopenshift20230401.OpenShiftClusterProperties{
				ResourceTags: testResourceTags,
			},
		}

		oc, err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, updateParameters)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(policyAssignmentLagDuration)

		By("verifying the request to replace the set of tags fully replaced the previous set stored in the API field")
		Expect(oc.OpenShiftClusterProperties.ResourceTags).To(Equal(testResourceTags))

		By("verifying the new set of tags is present on the resource group")
		group, err = clients.ResourceGroups.Get(ctx, clusterResourceGroupID)
		Expect(err).NotTo(HaveOccurred())

		for k, v := range testResourceTags {
			Expect(group.Tags).Should(HaveKey(k))
			Expect(*group.Tags[k]).To(Equal(*v))
		}

		By("verifying the new set of tags is present on the existing resources")
		resources, err := clients.Resources.ListByResourceGroup(ctx, clusterResourceGroupID, "", "", nil)
		Expect(err).NotTo(HaveOccurred())

		for _, r := range resources {
			for k, v := range testResourceTags {
				Expect(r.Tags).Should(HaveKey(k))
				Expect(*r.Tags[k]).To(Equal(*v))
			}
		}

		// Creating an arbitrary Azure resource accomplishes the same thing as a more complex test that would trigger OCP
		// to create an Azure resource. We just want to see that new resources provisioned within the cluster
		// resource group have the tags placed on them by the policy.
		By("creating a new NSG within the cluster RG and verifying the tagging policy assigns the new set of tags")
		nsgCreateParameters := mgmtnetwork.SecurityGroup{
			Location:                      oc.Location,
			SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
		}

		err = clients.NetworkSecurityGroups.CreateOrUpdateAndWait(ctx, clusterResourceGroupID, nsgName, nsgCreateParameters)
		Expect(err).NotTo(HaveOccurred())

		// The tagging policy's modify effect takes just a little bit longer
		// than the create operation.
		time.Sleep(30 * time.Second)

		nsg, err := clients.NetworkSecurityGroups.Get(ctx, clusterResourceGroupID, nsgName, "")
		Expect(err).NotTo(HaveOccurred())

		for k, v := range testResourceTags {
			Expect(nsg.Tags).Should(HaveKey(k))
			Expect(*nsg.Tags[k]).To(Equal(*v))
		}

		By("replacing the current set of tags with the empty set")
		updateParameters = mgmtredhatopenshift20230401.OpenShiftClusterUpdate{
			OpenShiftClusterProperties: &mgmtredhatopenshift20230401.OpenShiftClusterProperties{
				ResourceTags: map[string]*string{},
			},
		}

		oc, err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, updateParameters)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(policyAssignmentLagDuration)

		By("verifying that the API field now contains the empty set")
		Expect(oc.OpenShiftClusterProperties.ResourceTags).To(Equal(map[string]*string{}))

		By("verifying that the tags are still present on the resource group despite being removed from the API field")
		group, err = clients.ResourceGroups.Get(ctx, clusterResourceGroupID)
		Expect(err).NotTo(HaveOccurred())

		for k, v := range testResourceTags {
			Expect(group.Tags).Should(HaveKey(k))
			Expect(*group.Tags[k]).To(Equal(*v))
		}

		By("verifying that the tags are still present on the resources despite being removed from the API field")
		resources, err = clients.Resources.ListByResourceGroup(ctx, clusterResourceGroupID, "", "", nil)
		Expect(err).NotTo(HaveOccurred())

		for _, r := range resources {
			for k, v := range testResourceTags {
				Expect(r.Tags).Should(HaveKey(k))
				Expect(*r.Tags[k]).To(Equal(*v))
			}
		}
	})
})

func restoreClusterResourceTags(ctx context.Context) (oc mgmtredhatopenshift20230401.OpenShiftCluster, group mgmtfeatures.ResourceGroup, err error) {
	updateParameters := mgmtredhatopenshift20230401.OpenShiftClusterUpdate{
		OpenShiftClusterProperties: &mgmtredhatopenshift20230401.OpenShiftClusterProperties{
			ResourceTags: originalResourceTags,
		},
	}

	oc, err = clients.OpenshiftClusters.UpdateAndWait(ctx, vnetResourceGroup, clusterName, updateParameters)
	if err != nil {
		return mgmtredhatopenshift20230401.OpenShiftCluster{}, mgmtfeatures.ResourceGroup{}, err
	}

	time.Sleep(policyAssignmentLagDuration)

	group, err = clients.ResourceGroups.Get(ctx, clusterResourceGroupID)
	if err != nil {
		return mgmtredhatopenshift20230401.OpenShiftCluster{}, mgmtfeatures.ResourceGroup{}, err
	}

	group.Tags = originalResourceTagsAzure

	group, err = clients.ResourceGroups.CreateOrUpdate(ctx, clusterResourceGroupID, group)
	if err != nil {
		return mgmtredhatopenshift20230401.OpenShiftCluster{}, mgmtfeatures.ResourceGroup{}, err
	}

	resources, err := clients.Resources.ListByResourceGroup(ctx, clusterResourceGroupID, "", "", nil)
	Expect(err).NotTo(HaveOccurred())

	for _, r := range resources {
		if r.Tags == nil {
			r.Tags = map[string]*string{}
		}

		for k := range testResourceTags {
			delete(r.Tags, k)
		}

		parameters := mgmtfeatures.GenericResource{
			Tags: r.Tags,
		}

		err = clients.Resources.UpdateByIDAndWait(ctx, *r.ID, "2021-04-01", parameters)
		Expect(err).NotTo(HaveOccurred())
	}

	return oc, group, nil
}
