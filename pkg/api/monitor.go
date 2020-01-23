package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Monitor represents a monitor
type Monitor struct {
	MissingFields

	Buckets []string `json:"buckets,omitempty"`
}
