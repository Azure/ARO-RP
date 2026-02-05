package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"slices"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
)

type SelectorDataType string

const (
	SelectorDataTypeString SelectorDataType = "string"
	SelectorDataTypeDate   SelectorDataType = "date"
)

type SelectorDataKey string

const (
	SelectorDataKeyResourceID        SelectorDataKey = "resourceID"
	SelectorDataKeySubscriptionID    SelectorDataKey = "subscriptionID"
	SelectorDataKeySubscriptionState SelectorDataKey = "subscriptionState"
)

type selectorData map[SelectorDataKey]string

func (s selectorData) GetType(key string) SelectorDataType {
	switch SelectorDataKey(key) {
	case SelectorDataKeyResourceID, SelectorDataKeySubscriptionID, SelectorDataKeySubscriptionState:
		return SelectorDataTypeString
	}

	return SelectorDataTypeString
}

func (s selectorData) GetString(key string) (string, bool) {
	val, ok := s[SelectorDataKey(key)]
	return val, ok
}

func (s selectorData) Matches(log *logrus.Entry, selectors []*api.MaintenanceScheduleSelector) (bool, error) {
	matches := true

	// Empty selector list never matches
	if len(selectors) == 0 {
		return false, errors.New("empty selector list")
	}

	for _, selector := range selectors {
		selectorType := s.GetType(selector.Key)

		switch selectorType {
		case SelectorDataTypeString:
			selectorVal, exists := s.GetString(selector.Key)
			if !exists {
				// selector doesn't match key
				return false, fmt.Errorf("requested non-existent '%s' selector key", selector.Key)
			}
			switch selector.Operator {
			// string eq
			case api.MaintenanceScheduleSelectorOperatorEq:
				if selectorVal != selector.Value {
					// does not match
					matches = false
				}

			// string in/not-in
			case api.MaintenanceScheduleSelectorOperatorIn, api.MaintenanceScheduleSelectorOperatorNotIn:
				contains := slices.Contains(selector.Values, selectorVal)

				if (!contains && selector.Operator == api.MaintenanceScheduleSelectorOperatorIn) ||
					(contains && selector.Operator == api.MaintenanceScheduleSelectorOperatorNotIn) {
					matches = false
				}
			default:
				return false, fmt.Errorf("unknown selector operator %s", selector.Operator)
			}
		default:
			return false, fmt.Errorf("unhandled type '%s' of key '%s'", selectorType, selector.Key)
		}
	}
	return matches, nil
}
