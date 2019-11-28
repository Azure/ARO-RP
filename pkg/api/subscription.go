package api

// Subscription represents a subscription
type Subscription struct {
	MissingFields

	State SubscriptionState `json:"state,omitempty"`
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
