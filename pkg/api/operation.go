package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OperationList represents an RP operation list.
type OperationList struct {
	// List of operations supported by the resource provider.
	Operations []Operation `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// Operation represents an RP operation.
type Operation struct {
	// Operation name: {provider}/{resource}/{operation}.
	Name string `json:"name,omitempty"`

	// The object that describes the operation.
	Display Display `json:"display,omitempty"`

	// Sources of requests to this operation.  Comma separated list with valid values user or system, e.g. "user,system".
	Origin string `json:"origin,omitempty"`
}

// Display represents the display details of an operation.
type Display struct {
	// Friendly name of the resource provider.
	Provider string `json:"provider,omitempty"`

	// Resource type on which the operation is performed.
	Resource string `json:"resource,omitempty"`

	// Operation type: read, write, delete, listKeys/action, etc.
	Operation string `json:"operation,omitempty"`

	// Friendly name of the operation.
	Description string `json:"description,omitempty"`
}
