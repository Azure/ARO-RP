package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Subscription represents a subscription
type Subscription struct {
	MissingFields

	State SubscriptionState `json:"state,omitempty"`

	Properties *SubscriptionProperties `json:"properties,omitempty"`
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
