package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"
)

// AsyncOperation represents an asyncOperation
type AsyncOperation struct {
	MissingFields

	ID   string `json:"id,omitempty" deep:"-"`
	Name string `json:"name,omitempty" deep:"-"`

	InitialProvisioningState ProvisioningState `json:"initialStatus,omitempty"`
	ProvisioningState        ProvisioningState `json:"status,omitempty"`

	StartTime time.Time  `json:"startTime,omitempty" deep:"-"`
	EndTime   *time.Time `json:"endTime,omitempty" deep:"-"`

	Error *CloudErrorBody `json:"error,omitempty"`
}
