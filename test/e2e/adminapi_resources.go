package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var _ = Describe("[Admin API] List Azure resources action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should list Azure resources", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()

		By("getting the resource group where cluster resources live in")
		oc, err := clients.OpenshiftClusters.Get(ctx, im.ResourceGroup(), clusterName)
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
		vnetName, _, err := subnet.Split(*oc.OpenShiftClusterProperties.MasterProfile.SubnetID)
		Expect(err).NotTo(HaveOccurred())
		expectedResourceIDs = append(expectedResourceIDs, strings.ToLower(vnetName))

		By("getting the actual Azure resource IDs via admin actions API")
		var actualResources []mgmtfeatures.GenericResourceExpanded
		resp, err := adminRequest(ctx, http.MethodGet, "/admin"+resourceID+"/resources", nil, nil, &actualResources)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("reading response")
		actualResourceIDs := make([]string, 0, len(actualResources))
		for _, r := range actualResources {
			id := strings.ToLower(*r.ID)

			// HACK: exclude route tables from the comparison for now.
			if strings.Contains(id, "/providers/microsoft.network/routetables/") {
				continue
			}

			actualResourceIDs = append(actualResourceIDs, id)
		}

		By("comparing lists of resources")
		Expect(actualResourceIDs).To(ConsistOf(expectedResourceIDs))
	})
})
