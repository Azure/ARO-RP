package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/mimo"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

var _ = Describe("MIMO Actuator E2E Testing", Serial, func() {
	BeforeEach(func() {
		skipIfNotInDevelopmentEnv()
		skipIfMIMOActuatorNotEnabled()

		DeferCleanup(func(ctx context.Context) {
			// reset feature flags to their default values
			var oc = &admin.OpenShiftCluster{}
			resp, err := adminRequest(ctx,
				http.MethodPatch, clusterResourceID, nil, true,
				json.RawMessage("{\"operatorFlagsMergeStrategy\": \"reset\", \"properties\": {\"maintenanceTask\": \"SyncClusterObject\"}}"), oc)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Wait for the flag reset to finish applying
			Eventually(func(g Gomega, ctx context.Context) {
				oc = adminGetCluster(g, ctx, clusterResourceID)
				g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
			}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())
		})
	})

	It("Should be able to schedule and run a maintenance set via the admin API", func(ctx context.Context) {
		var oc = &admin.OpenShiftCluster{}
		testflag := "aro.e2e.testflag." + uuid.DefaultGenerator.Generate()

		By("set a bogus flag on the cluster")
		resp, err := adminRequest(ctx,
			http.MethodPatch, clusterResourceID, nil, true,
			json.RawMessage("{\"properties\": {\"maintenanceTask\": \"SyncClusterObject\", \"operatorFlags\": {\""+testflag+"\": \"true\"}}}"), oc)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		By("waiting for the update to complete")
		Eventually(func(g Gomega, ctx context.Context) {
			oc = adminGetCluster(g, ctx, clusterResourceID)
			g.Expect(oc.Properties.ProvisioningState).To(Equal(admin.ProvisioningStateSucceeded))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("check the flag is set in the cluster")
		co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		flag, ok := co.Spec.OperatorFlags[testflag]
		Expect(ok).To(BeTrue())
		Expect(flag).To(Equal("true"))

		By("change the flag in-cluster to a wrong value")
		// get the flag we want to check for
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			co, err := clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
			if err != nil {
				return err
			}

			co.Spec.OperatorFlags[testflag] = operator.FlagFalse
			_, err = clients.AROClusters.AroV1alpha1().Clusters().Update(ctx, co, metav1.UpdateOptions{})
			return err
		})
		Expect(err).NotTo(HaveOccurred())

		By("creating the flag update manifest via the API")
		out := &admin.MaintenanceManifest{}
		resp, err = adminRequest(ctx,
			http.MethodPut, "/admin"+clusterResourceID+"/maintenancemanifests",
			url.Values{}, true, &admin.MaintenanceManifest{
				MaintenanceTaskID: mimo.OPERATOR_FLAGS_UPDATE_ID,
			}, &out, logOnError(log)...)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))

		manifestID := out.ID

		By("waiting for the manifest run to complete")
		Eventually(func(g Gomega, ctx context.Context) {
			fetchedManifest := &admin.MaintenanceManifest{}
			resp, err = adminRequest(ctx,
				http.MethodGet, "/admin"+clusterResourceID+"/maintenancemanifests/"+manifestID,
				url.Values{}, true, nil, &fetchedManifest)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
			g.Expect(fetchedManifest.State).To(Equal(admin.MaintenanceManifestStateCompleted))
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("checking the flag has been set back in the cluster")
		co, err = clients.AROClusters.AroV1alpha1().Clusters().Get(ctx, "cluster", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		flag, ok = co.Spec.OperatorFlags[testflag]
		Expect(ok).To(BeTrue())
		Expect(flag).To(Equal("true"), "MIMO manifest has not run")
	})
})
