package api

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
	Error  error            `json:"error,omitempty"`
}

// TypeMeta describes an individual API model object
type TypeMeta struct {
	// APIVersion is on every object
	APIVersion string `json:"apiVersion"`
}

// ResourceTypeMeta is the Typemeta inside request body of preflight
type ResourceTypeMeta struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	Properties `json:"properties"`
	TypeMeta
}

type Properties map[string]interface{}

type ValidationStatus string

const (
	// ValidationStatusSucceeded means validation passed
	ValidationStatusSucceeded ValidationStatus = "Succeeded"
	// ValidationStatusFailed means validation failed
	ValidationStatusFailed ValidationStatus = "Failed"
)
