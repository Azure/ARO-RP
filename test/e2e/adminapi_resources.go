package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var _ = Describe("[Admin API] List Azure resources action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should list Azure resources", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		By("getting the resource group where cluster resources live in")
		oc, err := clients.OpenshiftClustersv20200430.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("building a list of valid Azure resource IDs via the Azure API")
		expectedResources, err := clients.Resources.ListByResourceGroup(ctx, clusterResourceGroup, "", "", nil)
		Expect(err).NotTo(HaveOccurred())

		expectedResourceIDs := make([]string, 0, len(expectedResources)+1)
		for _, r := range expectedResources {
			expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(*r.ID))
		}

		By("adding VNet to list of valid Azure resource IDs")
		vnetID, _, err := subnet.Split(*oc.OpenShiftClusterProperties.MasterProfile.SubnetID)
		Expect(err).NotTo(HaveOccurred())
		expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(vnetID))

		By("adding RouteTables to list of valid Azure resource IDs")
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

		diskEncryptionSet, err := clients.DiskEncryptionSets.Get(ctx, vnetResourceGroup, fmt.Sprintf("%s-disk-encryption-set", vnetResourceGroup))
		Expect(err).NotTo(HaveOccurred())
		expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(*diskEncryptionSet.ID))

		for _, subnet := range *vnet.Subnets {
			if _, ok := subnets[strings.ToLower(*subnet.ID)]; !ok {
				continue
			}

			if subnet.SubnetPropertiesFormat != nil &&
				subnet.RouteTable != nil {
				expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(*subnet.RouteTable.ID))
			}
		}

		By("getting the actual Azure resource IDs via admin actions API")
		var actualResources []mgmtfeatures.GenericResourceExpanded
		resp, err := adminRequest(ctx, http.MethodGet, "/admin"+resourceID+"/resources", nil, nil, &actualResources)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("reading response")
		actualResourceIDs := make([]string, 0, len(actualResources))
		for _, r := range actualResources {
			id := strings.ToLower(*r.ID)
			actualResourceIDs = append(actualResourceIDs, id)
		}

		By("comparing lists of resources")
		Expect(actualResourceIDs).To(ConsistOf(expectedResourceIDs))
	})
})
