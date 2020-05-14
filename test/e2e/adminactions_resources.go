package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

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

		// Get Azure resources via admin actions API
		_, err = adminRequest("GET", "resources", "", nil)
		Expect(err).NotTo(HaveOccurred())

		// Get Azure resources via Azure API
		clusterRG := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')
		resources, err := Clients.Resources.ListByResourceGroup(ctx, clusterRG, "", "", nil)
		Expect(err).NotTo(HaveOccurred())
		Log.Debugln(resources)
	})
})
