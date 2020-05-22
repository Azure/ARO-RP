package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var _ = Describe("Admin actions", func() {
	BeforeEach(runAdminTestsInDevOnly)

	Specify("List Azure resources", func() {
		ctx := context.Background()

		oc, err := Clients.OpenshiftClusters.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("CLUSTER"))
		Expect(err).NotTo(HaveOccurred())

		// Build a list of valid Azure resource ID's via the Azure API
		clusterRG := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')
		resources, err := Clients.Resources.ListByResourceGroup(ctx, clusterRG, "", "", nil)
		Expect(err).NotTo(HaveOccurred())
		var resourceIDs []string
		for _, r := range resources {
			resourceIDs = append(resourceIDs, strings.ToLower(*r.ID))
		}

		// Get Azure resource names via admin actions API
		result, err := adminRequest("GET", "resources", "", nil)
		Expect(err).NotTo(HaveOccurred())

		// Unmarshal the JSON and assert the ID's are expected
		var data []map[string]interface{}
		err = json.Unmarshal(result, &data)
		Expect(err).NotTo(HaveOccurred())
		for _, resource := range data {
			Expect(strings.ToLower(fmt.Sprintf("%s", resource["id"]))).To(BeElementOf(resourceIDs))
		}
	})
})
