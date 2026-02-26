package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	testdatabase "github.com/Azure/ARO-RP/test/database"
)

func TestMIMOListSchedules(t *testing.T) {
	ctx := context.Background()

	type test struct {
		name           string
		fixtures       func(f *testdatabase.Fixture)
		limit          int
		wantStatusCode int
		wantResponse   *admin.MaintenanceScheduleList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:     "no entries",
			fixtures: func(f *testdatabase.Fixture) {},
			wantResponse: &admin.MaintenanceScheduleList{
				MaintenanceSchedules: []*admin.MaintenanceSchedule{},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "single entry",
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
			wantResponse: &admin.MaintenanceScheduleList{
				MaintenanceSchedules: []*admin.MaintenanceSchedule{
					{
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
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:  "limit over",
			limit: 1,
			fixtures: func(f *testdatabase.Fixture) {
				f.AddMaintenanceScheduleDocuments(
					&api.MaintenanceScheduleDocument{
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
					},
					&api.MaintenanceScheduleDocument{
						ID: "08080808-0808-0808-0808-080808080002",
						MaintenanceSchedule: api.MaintenanceSchedule{
							MaintenanceTaskID: "exampletask2",
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
					},
				)
			},
			wantResponse: &admin.MaintenanceScheduleList{
				NextLink: "https://mockrefererhost/?%24skipToken=" + url.QueryEscape(base64.StdEncoding.EncodeToString([]byte("FAKE1"))),
				MaintenanceSchedules: []*admin.MaintenanceSchedule{
					{
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
				},
			},
			wantStatusCode: http.StatusOK,
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

			if tt.limit == 0 {
				tt.limit = 100
			}

			fmt.Printf("limit: %d", tt.limit)

			resp, b, err := ti.request(http.MethodGet,
				fmt.Sprintf("https://server/admin/maintenanceschedules?limit=%d", tt.limit),
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
