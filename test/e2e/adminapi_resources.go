package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var _ = Describe("[Admin API] List Azure resources action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must list Azure resources for a cluster", func(ctx context.Context) {
		By("getting the resource group where cluster resources live in")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("getting a list of resources from the cluster resource group via ARM")
		expectedResources, err := clients.Resources.ListByResourceGroup(ctx, clusterResourceGroup, "", "", nil)
		Expect(err).NotTo(HaveOccurred())

		By("building a list of expected Azure resource IDs")
		expectedResourceIDs := make([]string, 0, len(expectedResources)+1)
		for _, r := range expectedResources {
			expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(*r.ID))
		}

		By("adding disk encryption sets to the the list of expected resource IDs")
		diskEncryptionSet, err := clients.DiskEncryptionSets.Get(ctx, vnetResourceGroup, fmt.Sprintf("%s-disk-encryption-set", vnetResourceGroup))
		Expect(err).NotTo(HaveOccurred())
		expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(*diskEncryptionSet.ID))

		By("adding VNet to the list of expected resource IDs")
		vnetID, _, err := subnet.Split(*oc.OpenShiftClusterProperties.MasterProfile.SubnetID)
		Expect(err).NotTo(HaveOccurred())
		expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(vnetID))

		By("adding RouteTables to the list of expected resource IDs")
		r, err := azure.ParseResourceID(vnetID)
		Expect(err).NotTo(HaveOccurred())

		subnets := map[string]struct{}{
			strings.ToLower(*oc.OpenShiftClusterProperties.MasterProfile.SubnetID): {},
		}
		for _, p := range *oc.OpenShiftClusterProperties.WorkerProfiles {
			subnets[strings.ToLower(*p.SubnetID)] = struct{}{}
		}

		vnet, err := clients.VirtualNetworks.Get(ctx, r.ResourceGroup, r.ResourceName, "")
		Expect(err).NotTo(HaveOccurred())

		for _, subnet := range *vnet.Subnets {
			if _, ok := subnets[strings.ToLower(*subnet.ID)]; !ok {
				continue
			}

			if subnet.SubnetPropertiesFormat != nil &&
				subnet.RouteTable != nil {
				expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(*subnet.RouteTable.ID))
			}
		}

		By("getting the actual Azure resource IDs via RP admin API")
		var actualResources []mgmtfeatures.GenericResourceExpanded
		// Don't strictly check for unknown fields because the upstream struct doesn't account for an Etag
		resp, err := adminRequest(ctx, http.MethodGet, "/admin"+clusterResourceID+"/resources", nil, false, nil, &actualResources)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("reading response")
		actualResourceIDs := make([]string, 0, len(actualResources))
		for _, r := range actualResources {
			id := strings.ToLower(*r.ID)
			actualResourceIDs = append(actualResourceIDs, id)
		}

		By("verifying the list of resources")
		Expect(actualResourceIDs).To(ConsistOf(expectedResourceIDs))
	})
})
