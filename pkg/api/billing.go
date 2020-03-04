package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"
)

// Billing represents a Billing entry
type Billing struct {
	MissingFields

	CreationTime    time.Time  `json:"creationTime,omitempty"`
	DeletionTime    *time.Time `json:"deletionTime,omitempty"`
	LastBillingTime time.Time  `json:"lastBillingTime,omitempty"`
}
