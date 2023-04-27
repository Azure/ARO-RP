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
	Status ValidationStatus            `json:"status"`
	Error  *ManagementErrorWithDetails `json:"error,omitempty"`
}

type ManagementErrorWithDetails struct {
	// Code - The error code returned from the server.
	Code *string `json:"code,omitempty"`
	// Message - The error message returned from the server.
	Message *string `json:"message,omitempty"`
	// Target - The target of the error.
	Target *string `json:"target,omitempty"`
	// Details - Validation error.
	Details *[]ManagementErrorWithDetails `json:"details,omitempty"`
}

// ResourceTypeMeta is the Typemeta inside request body of preflight
type ResourceTypeMeta struct {
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
