package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
)

// PreflightRequest is the request body of preflight
type PreflightRequest struct {
	Resources []json.RawMessage `json:"resources"`
}

// ValidationResult is the validation result to return in the deployment preflight response body
type ValidationResult struct {
	Status ValidationStatus `json:"status"`
	Error  *CloudErrorBody  `json:"error,omitempty"`
}

// ResourceTypeMeta is the Typemeta inside request body of preflight
type ResourceTypeMeta struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	APIVersion string `json:"apiVersion"`
	Properties `json:"properties"`
}

type Properties map[string]interface{}

type ValidationStatus string

const (
	// ValidationStatusSucceeded means validation passed
	ValidationStatusSucceeded ValidationStatus = "Succeeded"
	// ValidationStatusFailed means validation failed
	ValidationStatusFailed ValidationStatus = "Failed"
)
