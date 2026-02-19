package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
)

type maintenanceScheduleStaticValidator struct{}

// Validate validates a MaintenanceSchedule
func (sv maintenanceScheduleStaticValidator) Static(_new interface{}, _current *api.MaintenanceScheduleDocument) error {
	new := _new.(*MaintenanceSchedule)

	var current *MaintenanceSchedule
	if _current != nil {
		current = (&maintenanceScheduleConverter{}).ToExternal(_current).(*MaintenanceSchedule)
	}

	err := sv.validate(new)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return sv.validateDelta(new, current)
}

func (sv maintenanceScheduleStaticValidator) validate(new *MaintenanceSchedule) error {
	if new.MaintenanceTaskID == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "maintenanceTaskID", "Must be provided")
	}

	if new.LookForwardCount < 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "LookForwardCount", "Must be above 0")
	}

	if new.Schedule == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "schedule", "Must be provided")
	}

	if new.ScheduleAcross == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "scheduleAcross", "Must be provided")
	} else {
		_, err := time.ParseDuration(new.ScheduleAcross)
		if err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "scheduleAcross", fmt.Sprintf("Must be a valid time.Duration: %s", strings.TrimPrefix(err.Error(), "time: ")))
		}
	}

	if len(new.Selectors) == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "selectors", "Must be provided")
	}

	validOps := validSelectorOperators()
	for i, s := range new.Selectors {
		if !slices.Contains(validOps, s.Operator) {
			r := []string{}
			for _, v := range validOps {
				r = append(r, "'"+string(v)+"'")
			}

			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("selectors[%d].operator", i), fmt.Sprintf("Must be one of [%s]", strings.Join(r, ", ")))
		}
		if s.Operator == MaintenanceScheduleSelectorOperatorIn || s.Operator == MaintenanceScheduleSelectorOperatorNotIn {
			if len(s.Values) == 0 {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("selectors[%d].values", i), fmt.Sprintf("Must be provided for operator type '%s'", s.Operator))
			}
			if s.Value != "" {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("selectors[%d].value", i), fmt.Sprintf("Must not be provided for operator type '%s'", s.Operator))
			}
		} else {
			if s.Value == "" {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("selectors[%d].value", i), fmt.Sprintf("Must be provided for operator type '%s'", s.Operator))
			}
			if len(s.Values) > 0 {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("selectors[%d].values", i), fmt.Sprintf("Must not be provided for operator type '%s'", s.Operator))
			}
		}
	}

	return nil
}

func (sv maintenanceScheduleStaticValidator) validateDelta(new, current *MaintenanceSchedule) error {
	err := immutable.Validate("", new, current)
	if err != nil {
		err := err.(*immutable.ValidationError)
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, err.Target, err.Message)
	}
	return nil
}
