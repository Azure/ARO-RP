package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/dns"
)

type SelectorDataType string

const (
	SelectorDataTypeString SelectorDataType = "string"
	SelectorDataTypeDate   SelectorDataType = "date"
)

type SelectorDataKey string

const (
	SelectorDataKeyResourceID         SelectorDataKey = "resourceID"
	SelectorDataKeySubscriptionID     SelectorDataKey = "subscriptionID"
	SelectorDataKeySubscriptionState  SelectorDataKey = "subscriptionState"
	SelectorDataKeyAuthenticationType SelectorDataKey = "authenticationType"
	SelectorDataArchitectureVersion   SelectorDataKey = "architectureVersion"
	SelectorDataProvisioningState     SelectorDataKey = "provisioningState"
	SelectorDataOutboundType          SelectorDataKey = "outboundType"
	SelectorDataAPIServerVisibility   SelectorDataKey = "APIServerVisibility"
	SelectorDataIsManagedDomain       SelectorDataKey = "isManagedDomain"
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

func ToSelectorData(doc *api.OpenShiftClusterDocument, subscriptionState string) (selectorData, error) {
	new := selectorData{}

	resourceID := strings.ToLower(doc.OpenShiftCluster.ID)
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}

	new[SelectorDataKeyResourceID] = resourceID
	new[SelectorDataKeySubscriptionID] = r.SubscriptionID
	new[SelectorDataKeySubscriptionState] = subscriptionState
	if doc.OpenShiftCluster.UsesWorkloadIdentity() {
		new[SelectorDataKeyAuthenticationType] = "WorkloadIdentity"
	} else {
		new[SelectorDataKeyAuthenticationType] = "ServicePrincipal"
	}
	new[SelectorDataArchitectureVersion] = fmt.Sprintf("%d", doc.OpenShiftCluster.Properties.ArchitectureVersion)
	new[SelectorDataProvisioningState] = string(doc.OpenShiftCluster.Properties.ProvisioningState)
	new[SelectorDataOutboundType] = string(doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType)
	new[SelectorDataAPIServerVisibility] = string(doc.OpenShiftCluster.Properties.APIServerProfile.Visibility)
	new[SelectorDataIsManagedDomain] = fmt.Sprintf("%t", dns.IsManagedDomain(doc.OpenShiftCluster.Properties.ClusterProfile.Domain))
	return new, nil
}
