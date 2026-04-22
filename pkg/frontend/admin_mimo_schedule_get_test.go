package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestMIMOGetSchedule(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		scheduleID     string
		fixtures       func(f *testdatabase.Fixture)
		wantStatusCode int
		wantResponse   *admin.MaintenanceSchedule
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "single entry",
			scheduleID: "08080808-0808-0808-0808-080808080001",
			fixtures: func(f *testdatabase.Fixture) {
				f.AddMaintenanceScheduleDocuments(&api.MaintenanceScheduleDocument{
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
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "not found",
			scheduleID:     "something",
			wantStatusCode: http.StatusNotFound,
			wantError:      "404: NotFound: : schedule 'something' not found: 404 : ",
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

			f, err := NewFrontend(ctx, ti.auditLog, ti.log, ti.otelAudit, ti.env, ti.dbGroup, api.APIs, &noop.Noop{}, &noop.Noop{}, testdatabase.NewFakeAEAD(), nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/admin/maintenanceschedules/%s", tt.scheduleID),
				http.Header{
					"Referer": []string{"https://mockrefererhost/"},
				}, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
