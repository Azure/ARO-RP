package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Query represents a query
type Query struct {
	Query      string      `json:"query,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

// Parameter represents a parameter
type Parameter struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}
