package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaintenanceScheduleStaticValidator_validate(t *testing.T) {
	tests := []struct {
		name    string
		new     *MaintenanceSchedule
		wantErr string
	}{
		{
			name: "valid case",
			new: &MaintenanceSchedule{
				ID:                "00000",
				MaintenanceTaskID: MIMOTaskID("0"),
				State:             MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				ScheduleAcross:    "12h",
				Selectors: []*MaintenanceScheduleSelector{
					{
						Key:      "something",
						Operator: MaintenanceScheduleSelectorOperatorIn,
						Values:   []string{"foobar"},
					},
				},
			},
		},
		{
			name: "selector-in with value, not values",
			new: &MaintenanceSchedule{
				ID:                "00000",
				MaintenanceTaskID: MIMOTaskID("0"),
				State:             MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				ScheduleAcross:    "12h",
				Selectors: []*MaintenanceScheduleSelector{
					{
						Key:      "something",
						Operator: MaintenanceScheduleSelectorOperatorIn,
						Value:    "foobar",
					},
				},
			},
			wantErr: "400: InvalidParameter: selectors[0].values: Must be provided for operator type 'in'",
		},
		{
			name: "selector-eq with values, not value",
			new: &MaintenanceSchedule{
				ID:                "00000",
				MaintenanceTaskID: MIMOTaskID("0"),
				State:             MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				ScheduleAcross:    "12h",
				Selectors: []*MaintenanceScheduleSelector{
					{
						Key:      "something",
						Operator: MaintenanceScheduleSelectorOperatorEq,
						Values:   []string{"foobar"},
					},
				},
			},
			wantErr: "400: InvalidParameter: selectors[0].value: Must be provided for operator type 'eq'",
		},
		{
			name: "selector with nonsense operator",
			new: &MaintenanceSchedule{
				ID:                "00000",
				MaintenanceTaskID: MIMOTaskID("0"),
				State:             MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				ScheduleAcross:    "12h",
				Selectors: []*MaintenanceScheduleSelector{
					{
						Key:      "something",
						Operator: MaintenanceScheduleSelectorOperator("baz"),
						Value:    "foobar",
					},
				},
			},
			wantErr: "400: InvalidParameter: selectors[0].operator: Must be one of ['eq', 'in', 'notin']",
		},
		{
			name: "missing scheduleacross",
			new: &MaintenanceSchedule{
				ID:                "00000",
				MaintenanceTaskID: MIMOTaskID("0"),
				State:             MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				Selectors: []*MaintenanceScheduleSelector{
					{
						Key:      "something",
						Operator: MaintenanceScheduleSelectorOperatorIn,
						Values:   []string{"foobar"},
					},
				},
			},
			wantErr: "400: InvalidParameter: scheduleAcross: Must be provided",
		},
		{
			name: "garbage scheduleacross",
			new: &MaintenanceSchedule{
				ID:                "00000",
				MaintenanceTaskID: MIMOTaskID("0"),
				State:             MaintenanceScheduleStateEnabled,
				Schedule:          "*-*-* 00:00:00",
				ScheduleAcross:    "1srgndf",
				Selectors: []*MaintenanceScheduleSelector{
					{
						Key:      "something",
						Operator: MaintenanceScheduleSelectorOperatorIn,
						Values:   []string{"foobar"},
					},
				},
			},
			wantErr: `400: InvalidParameter: scheduleAcross: Must be a valid time.Duration: unknown unit "srgndf" in duration "1srgndf"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv := maintenanceScheduleStaticValidator{}
			gotErr := sv.validate(tt.new)

			if tt.wantErr != "" {
				require.Equal(t, tt.wantErr, gotErr.Error())
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}
