package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"
)

// Billing represents a Billing entry
type Billing struct {
	MissingFields

	CreationTime    int        `json:"creationTime,omitempty"`
	DeletionTime    *time.Time `json:"deletionTime,omitempty"`
	LastBillingTime int        `json:"lastBillingTime,omitempty"`

	Location string `json:"location,omitempty"`
	TenantID string `json:"tenantID,omitempty"`
}
