package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/mimo"
	"github.com/Azure/ARO-RP/pkg/mimo/scheduler"
)

var _ = Describe("MIMO Scheduler E2E Testing", Serial, func() {
	BeforeEach(func() {
		skipIfNotInDevelopmentEnv()
		skipIfMIMONotEnabled()
	})

	It("Should be able to create a schedule which makes Manifests via the admin API", func(ctx context.Context) {
		// Truncate the hour and set it 2h in the future, we only want to see it
		// created, not run
		scheduleTime := time.Now().UTC().Truncate(time.Hour).Add(2 * time.Hour)

		By("creating the flag update manifest via the API")
		out := &admin.MaintenanceSchedule{}
		resp, err := adminRequest(ctx,
			http.MethodPut, "/admin/maintenanceschedules",
			url.Values{}, true, &admin.MaintenanceSchedule{
				State:             admin.MaintenanceScheduleStateEnabled,
				Schedule:          scheduleTime.Format(time.DateTime),
				ScheduleAcross:    "0h",
				MaintenanceTaskID: admin.MIMOTaskID(mimo.OPERATOR_FLAGS_UPDATE_ID),
				Selectors: []*admin.MaintenanceScheduleSelector{
					{
						Key:      string(scheduler.SelectorDataKeySubscriptionState),
						Operator: admin.MaintenanceScheduleSelectorOperatorIn,
						Values:   []string{string(api.SubscriptionStateRegistered)},
					},
				},
			}, &out, logOnError(log)...)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))

		scheduleID := out.ID

		By("waiting for the schedule to create tasks")
		Eventually(func(g Gomega, ctx context.Context) {
			fetchedManifests := &admin.MaintenanceManifestList{}
			resp, err = adminRequest(ctx,
				http.MethodGet, "/admin"+clusterResourceID+"/maintenancemanifests",
				url.Values{}, true, nil, &fetchedManifests)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))

			found := 0
			for _, manifest := range fetchedManifests.MaintenanceManifests {
				if manifest.CreatedBySchedule == admin.MIMOScheduleID(scheduleID) {
					g.Expect(string(manifest.MaintenanceTaskID)).To(Equal(string(mimo.OPERATOR_FLAGS_UPDATE_ID)))
					g.Expect(manifest.RunAfter).To(Equal(scheduleTime.Unix()))
					found = found + 1
				}
			}
			g.Expect(found).To(Equal(1), "expect 1 created manifest")
		}).WithContext(ctx).WithTimeout(DefaultEventuallyTimeout).Should(Succeed())

		By("disabling the schedule, tasks will not be recreated")
		out = &admin.MaintenanceSchedule{}
		resp, err = adminRequest(ctx,
			http.MethodPut, "/admin/maintenanceschedules",
			url.Values{}, true, &admin.MaintenanceSchedule{
				State:             admin.MaintenanceScheduleStateDisabled,
				Schedule:          scheduleTime.Format(time.DateTime),
				ScheduleAcross:    "0h",
				MaintenanceTaskID: admin.MIMOTaskID(mimo.OPERATOR_FLAGS_UPDATE_ID),
				Selectors: []*admin.MaintenanceScheduleSelector{
					{
						Key:      string(scheduler.SelectorDataKeySubscriptionState),
						Operator: admin.MaintenanceScheduleSelectorOperatorIn,
						Values:   []string{string(api.SubscriptionStateRegistered)},
					},
				},
			}, &out, logOnError(log)...)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		// Cancel tasks created by this schedule
		vals := url.Values{}
		vals.Add("scheduleID", scheduleID)
		resp, err = adminRequest(ctx,
			http.MethodDelete, "/admin/maintenancemanifests/cancel",
			vals, true, nil, nil, logOnError(log)...)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		// Make sure a new task isn't created, wait for a bit beyond the usual
		// polling time of the scheduler
		Consistently(func(g Gomega, ctx context.Context) {
			fetchedManifests := &admin.MaintenanceManifestList{}
			resp, err = adminRequest(ctx,
				http.MethodGet, "/admin"+clusterResourceID+"/maintenancemanifests",
				url.Values{}, true, nil, &fetchedManifests)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))

			found := 0
			for _, manifest := range fetchedManifests.MaintenanceManifests {
				if manifest.CreatedBySchedule == admin.MIMOScheduleID(scheduleID) && manifest.State != admin.MaintenanceManifestStateCancelled {
					found = found + 1
				}
			}
			g.Expect(found).To(BeZero())
		}, time.Second*50, time.Second*10).WithContext(ctx).Should(Succeed())
	})
})
