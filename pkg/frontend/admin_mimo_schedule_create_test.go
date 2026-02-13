package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestMIMOCreateSchedule(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		fixtures       func(f *testdatabase.Fixture)
		body           *admin.MaintenanceSchedule
		wantStatusCode int
		wantResponse   *admin.MaintenanceSchedule
		wantResult     func(f *testdatabase.Checker)
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "invalid",
			fixtures: func(f *testdatabase.Fixture) {
			},
			body:           &admin.MaintenanceSchedule{},
			wantError:      "400: InvalidParameter: maintenanceTaskID: Must be provided",
			wantStatusCode: http.StatusBadRequest,
		},

		{
			name:     "good",
			fixtures: func(f *testdatabase.Fixture) {},
			body: &admin.MaintenanceSchedule{
				MaintenanceTaskID: "exampletask",
				State:             admin.MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				LookForwardCount:  1,
				Selectors: []*admin.MaintenanceScheduleSelector{
					{
						Key:      "foobar",
						Operator: admin.MaintenanceScheduleSelectorOperatorIn,
						Values:   []string{"baz"},
					},
				},
			},
			wantResult: func(c *testdatabase.Checker) {
				c.AddMaintenanceScheduleDocuments(&api.MaintenanceScheduleDocument{
					ID: "08080808-0808-0808-0808-080808080001",
					MaintenanceSchedule: api.MaintenanceSchedule{
						MaintenanceTaskID: "exampletask",
						State:             api.MaintenanceScheduleStateEnabled,
						Schedule:          "*-*-* 00:00:00",
						LookForwardCount:  1,
						Selectors: []*api.MaintenanceScheduleSelector{
							{
								Key:      "foobar",
								Operator: api.MaintenanceScheduleSelectorOperatorIn,
								Values:   []string{"baz"},
							},
						},
					},
				})
			},
			wantResponse: &admin.MaintenanceSchedule{
				ID:                "08080808-0808-0808-0808-080808080001",
				MaintenanceTaskID: "exampletask",
				State:             admin.MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				LookForwardCount:  1,
				Selectors: []*admin.MaintenanceScheduleSelector{
					{
						Key:      "foobar",
						Operator: admin.MaintenanceScheduleSelectorOperatorIn,
						Values:   []string{"baz"},
					},
				},
			},
			wantStatusCode: http.StatusCreated,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			now := func() time.Time { return time.Unix(1000, 0) }

			ti := newTestInfra(t).WithMaintenanceSchedules(now)
			defer ti.done()

			err := ti.buildFixtures(tt.fixtures)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantResult != nil {
				tt.wantResult(ti.checker)
			}

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			f.now = now

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodPut,
				"https://server/admin/maintenanceschedules",
				http.Header{
					"Content-Type": []string{"application/json"},
				}, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			for _, err := range ti.checker.CheckMaintenanceSchedules(ti.maintenanceSchedulesClient) {
				t.Error(err)
			}
		})
	}
}
