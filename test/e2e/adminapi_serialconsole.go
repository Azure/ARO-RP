package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
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
		clusterResourceGroup := stringutils.LastTokenByte(*oc.OpenShiftClusterProperties.ClusterProfile.ResourceGroupID, '/')

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
		resp, err := adminRequest(ctx, http.MethodPost, "/admin"+clusterResourceID+"/serialconsole", url.Values{"vmName": []string{vm}}, true, nil, &logs)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("decoding the logs, we can see Linux serial console")
		foundLogs := false
		b64Reader := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(logs))
		scanner := bufio.NewScanner(b64Reader)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Red Hat Enterprise Linux CoreOS") {
				foundLogs = true
			}
		}
		Expect(scanner.Err()).NotTo(HaveOccurred())

		Expect(foundLogs).To(BeTrue(), "expected to find serial console logs")

	})
})
