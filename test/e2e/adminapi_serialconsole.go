package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var _ = Describe("[Admin API] VM serial console action", func() {
	BeforeEach(skipIfNotInDevelopmentEnv)

	It("must return the serial console logs", func(ctx context.Context) {
		By("getting the resource group where the VM instances live in")
		oc, err := clients.OpenshiftClusters.Get(ctx, vnetResourceGroup, clusterName)
		Expect(err).NotTo(HaveOccurred())
		clusterResourceGroup := stringutils.LastTokenByte(*oc.ClusterProfile.ResourceGroupID, '/')

		By("picking a master node to get logs from")
		vms, err := clients.VirtualMachines.List(ctx, clusterResourceGroup)
		Expect(err).NotTo(HaveOccurred())
		Expect(vms).NotTo(BeEmpty())

		var vm string

		for _, possibleVM := range vms {
			if strings.Contains(*possibleVM.Name, "-master-") {
				vm = *possibleVM.Name
			}
		}
		log.Infof("selected vm: %s", vm)

		var logs string

		By("querying the serial console API")
		resp, err := adminRequest(ctx, http.MethodGet, "/admin"+clusterResourceID+"/serialconsole", url.Values{"vmName": []string{vm}}, true, nil, &logs)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("we can see Linux serial console")
		Expect(logs).To(ContainSubstring("Red Hat Enterprise Linux CoreOS"))
	})
})
