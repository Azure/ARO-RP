package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

type maintenanceScheduleConverter struct{}

func (m maintenanceScheduleConverter) ToExternal(d *api.MaintenanceScheduleDocument) interface{} {
	return &MaintenanceSchedule{
		ID: d.ID,

		State:             MaintenanceScheduleState(d.MaintenanceSchedule.State),
		MaintenanceTaskID: MIMOTaskID(d.MaintenanceSchedule.MaintenanceTaskID),

		Schedule:         d.MaintenanceSchedule.Schedule,
		LookForwardCount: d.MaintenanceSchedule.LookForwardCount,
		ScheduleAcross:   d.MaintenanceSchedule.ScheduleAcross,

		Selectors: convertSelectorsToExternal(d.MaintenanceSchedule.Selectors),
	}
}

func (m maintenanceScheduleConverter) ToExternalList(docs []*api.MaintenanceScheduleDocument, nextLink string) interface{} {
	l := &MaintenanceScheduleList{
		MaintenanceSchedules: make([]*MaintenanceSchedule, 0, len(docs)),
		NextLink:             nextLink,
	}

	for _, doc := range docs {
		l.MaintenanceSchedules = append(l.MaintenanceSchedules, m.ToExternal(doc).(*MaintenanceSchedule))
	}

	return l
}

func (m maintenanceScheduleConverter) ToInternal(_i interface{}, out *api.MaintenanceScheduleDocument) {
	i := _i.(*MaintenanceSchedule)

	out.ID = i.ID
	out.MaintenanceSchedule.MaintenanceTaskID = api.MIMOTaskID(i.MaintenanceTaskID)
	out.MaintenanceSchedule.State = api.MaintenanceScheduleState(i.State)

	out.MaintenanceSchedule.Schedule = i.Schedule
	out.MaintenanceSchedule.ScheduleAcross = i.ScheduleAcross
	out.MaintenanceSchedule.LookForwardCount = i.LookForwardCount
	out.MaintenanceSchedule.Selectors = convertSelectorsToInternal(i.Selectors)
}

func convertSelectorsToExternal(s []*api.MaintenanceScheduleSelector) []*MaintenanceScheduleSelector {
	r := []*MaintenanceScheduleSelector{}

	for _, i := range s {
		r = append(r, &MaintenanceScheduleSelector{
			Key:      i.Key,
			Operator: MaintenanceScheduleSelectorOperator(i.Operator),
			Value:    i.Value,
			Values:   i.Values,
		})
	}
	return r
}

func convertSelectorsToInternal(s []*MaintenanceScheduleSelector) []*api.MaintenanceScheduleSelector {
	r := []*api.MaintenanceScheduleSelector{}

	for _, i := range s {
		r = append(r, &api.MaintenanceScheduleSelector{
			Key:      i.Key,
			Operator: api.MaintenanceScheduleSelectorOperator(i.Operator),
			Value:    i.Value,
			Values:   i.Values,
		})
	}
	return r
}
