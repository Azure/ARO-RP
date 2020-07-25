package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var _ = Describe("[Admin API] List Azure resources action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("should list Azure resources", func() {
		ctx := context.Background()
		resourceID := resourceIDFromEnv()
		resourceGroup := os.Getenv("RESOURCEGROUP")
		resourceName := os.Getenv("CLUSTER")

		By("getting the resource group where cluster resources live in")
		oc, err := clients.OpenshiftClusters.Get(ctx, resourceGroup, resourceName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

		By("building a list of valid Azure resource IDs via the Azure API")
		expectedResources, err := clients.Resources.ListByResourceGroup(ctx, clusterResourceGroup, "", "", nil)
		Expect(err).NotTo(HaveOccurred())

		By("adding VNet to list of valid Azure resource IDs")
		aroResourceGroup := strings.TrimPrefix(clusterResourceGroup, "aro-")
		vNet, err := clients.Resources.ListByResourceGroup(ctx, aroResourceGroup, "resourceType eq 'Microsoft.Network/virtualNetworks'", "", nil)
		Expect(err).NotTo(HaveOccurred())
		expectedResources = append(vNet, expectedResources...)

		expectedResourceIDs := make([]string, len(expectedResources))
		for i, r := range expectedResources {
			expectedResourceIDs[i] = strings.ToLower(*r.ID)
		}

		By("getting the actual Azure resource IDs via admin actions API")
		var actualResources []mgmtfeatures.GenericResourceExpanded
		resp, err := adminRequest(ctx, http.MethodGet, "/admin"+resourceID+"/resources", nil, nil, &actualResources)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("reading response")
		actualResourceIDs := make([]string, len(actualResources))
		for i, r := range actualResources {
			actualResourceIDs[i] = strings.ToLower(*r.ID)
		}

		By("comparing lists of resources")
		Expect(actualResourceIDs).To(ConsistOf(expectedResourceIDs))
	})
})
