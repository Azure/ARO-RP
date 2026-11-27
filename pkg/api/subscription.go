package api

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Subscription represents a subscription
type Subscription struct {
	MissingFields

	State SubscriptionState `json:"state,omitempty"`

	Properties *SubscriptionProperties `json:"properties,omitempty"`

	Overrides *SubscriptionOverrides `json:"overrides,omitempty"`
}

// SubscriptionState represents a subscription state
type SubscriptionState string

// SubscriptionState constants
const (
	SubscriptionStateRegistered   SubscriptionState = "Registered"
	SubscriptionStateUnregistered SubscriptionState = "Unregistered"
	SubscriptionStateWarned       SubscriptionState = "Warned"
	SubscriptionStateSuspended    SubscriptionState = "Suspended"
	SubscriptionStateDeleted      SubscriptionState = "Deleted"
)

// SubscriptionProperties represents subscription properties
type SubscriptionProperties struct {
	MissingFields

	TenantID           string                     `json:"tenantId,omitempty"`
	AccountOwner       *AccountOwnerProfile       `json:"accountOwner,omitempty"`
	RegisteredFeatures []RegisteredFeatureProfile `json:"registeredFeatures,omitempty"`
}

// AccountOwnerProfile represents the subscription account owner information
type AccountOwnerProfile struct {
	MissingFields

	Email string `json:"email,omitempty"`
}

// RegisteredFeatureProfile represents the features registered to the subscription
type RegisteredFeatureProfile struct {
	MissingFields

	Name  string `json:"name,omitempty"`
	State string `json:"state,omitempty"`
}

// SubscriptionOverrides represents ARO-specific overrides for this Subscription
// in this region
type SubscriptionOverrides struct {
	MissingFields

	InstallationZones []string `json:"installationZones,omitempty"`
}

func (s *SubscriptionOverrides) Validate() error {
	if !slices.Contains([]int{0, 1, 3, 4}, len(s.InstallationZones)) {
		return NewCloudError(
			http.StatusBadRequest,
			CloudErrorCodeInvalidParameter,
			"overrides.installationZones",
			`only 0 (no override), 1, 3, or 4 zones allowed`)
	}

	for i, zone := range s.InstallationZones {
		if strings.EqualFold(zone, "nonzonal") && len(s.InstallationZones) > 1 {
			return NewCloudError(
				http.StatusBadRequest,
				CloudErrorCodeInvalidParameter,
				fmt.Sprintf("overrides.installationZones[%d]", i),
				`"nonzonal" cannot be mixed with zones`)
		}
		if !slices.Contains([]string{"1", "2", "3", "4"}, zone) {
			return NewCloudError(
				http.StatusBadRequest,
				CloudErrorCodeInvalidParameter,
				fmt.Sprintf("overrides.installationZones[%d]", i),
				fmt.Sprintf(`"%s" is not a valid zone number (1-4)`, zone))
		}
	}

	return nil
}
