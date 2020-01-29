package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/client/services/preview/redhatopenshift/mgmt/2019-12-31-preview/redhatopenshift"
)

var _ = Describe("List operations", func() {
	Specify("the correct static operations are returned", func() {
		ctx := context.Background()

		opList, err := Clients.Operations.List(ctx)
		Expect(err).NotTo(HaveOccurred())

		l := &redhatopenshift.OperationList{
			Value: &[]redhatopenshift.Operation{
				{
					Name: to.StringPtr("Microsoft.RedHatOpenShift/locations/operationresults/read"),
					Display: &redhatopenshift.Display{
						Provider:  to.StringPtr("Azure Red Hat OpenShift"),
						Resource:  to.StringPtr("locations/operationresults"),
						Operation: to.StringPtr("Read operation results"),
					},
				},
				{
					Name: to.StringPtr("Microsoft.RedHatOpenShift/locations/operationsstatus/read"),
					Display: &redhatopenshift.Display{
						Provider:  to.StringPtr("Azure Red Hat OpenShift"),
						Resource:  to.StringPtr("locations/operationsstatus"),
						Operation: to.StringPtr("Read operations status"),
					},
				},
				{
					Name: to.StringPtr("Microsoft.RedHatOpenShift/openShiftClusters/read"),
					Display: &redhatopenshift.Display{
						Provider:  to.StringPtr("Azure Red Hat OpenShift"),
						Resource:  to.StringPtr("openShiftClusters"),
						Operation: to.StringPtr("Read OpenShift cluster"),
					},
				},
				{
					Name: to.StringPtr("Microsoft.RedHatOpenShift/openShiftClusters/write"),
					Display: &redhatopenshift.Display{
						Provider:  to.StringPtr("Azure Red Hat OpenShift"),
						Resource:  to.StringPtr("openShiftClusters"),
						Operation: to.StringPtr("Write OpenShift cluster"),
					},
				},
				{
					Name: to.StringPtr("Microsoft.RedHatOpenShift/openShiftClusters/delete"),
					Display: &redhatopenshift.Display{
						Provider:  to.StringPtr("Azure Red Hat OpenShift"),
						Resource:  to.StringPtr("openShiftClusters"),
						Operation: to.StringPtr("Delete OpenShift cluster"),
					},
				},
				{
					Name: to.StringPtr("Microsoft.RedHatOpenShift/openShiftClusters/listCredentials/action"),
					Display: &redhatopenshift.Display{
						Provider:  to.StringPtr("Azure Red Hat OpenShift"),
						Resource:  to.StringPtr("openShiftClusters"),
						Operation: to.StringPtr("Lists credentials of an OpenShift cluster"),
					},
				},
				{
					Name: to.StringPtr("Microsoft.RedHatOpenShift/operations/read"),
					Display: &redhatopenshift.Display{
						Provider:  to.StringPtr("Azure Red Hat OpenShift"),
						Resource:  to.StringPtr("operations"),
						Operation: to.StringPtr("Read operations"),
					},
				},
			},
		}

		Expect(opList.Value).To(Equal(l.Value))
	})
})
