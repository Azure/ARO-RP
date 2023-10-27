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
