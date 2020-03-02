package arm

import "encoding/json"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Template represents an ARM template
type Template struct {
	Schema         string                        `json:"$schema,omitempty"`
	APIProfile     string                        `json:"apiProfile,omitempty"`
	ContentVersion string                        `json:"contentVersion,omitempty"`
	Variables      map[string]interface{}        `json:"variables,omitempty"`
	Parameters     map[string]*TemplateParameter `json:"parameters,omitempty"`
	Functions      []interface{}                 `json:"functions,omitempty"`
	Resources      []*Resource                   `json:"resources,omitempty"`
	Outputs        map[string]*Output            `json:"outputs,omitempty"`
}

// TemplateParameter represents an ARM template parameter
type TemplateParameter struct {
	Type          string                 `json:"type,omitempty"`
	DefaultValue  interface{}            `json:"defaultValue,omitempty"`
	AllowedValues []interface{}          `json:"allowedValues,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	MinValue      int                    `json:"minValue,omitempty"`
	MaxValue      int                    `json:"maxValue,omitempty"`
	MinLength     int                    `json:"minLength,omitempty"`
	MaxLength     int                    `json:"maxLength,omitempty"`
}

// Resource represents an ARM template resource
type Resource struct {
	Resource interface{}

	Name       string                 `json:"name,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Condition  bool                   `json:"condition,omitempty"`
	APIVersion string                 `json:"apiVersion,omitempty"`
	DependsOn  []string               `json:"dependsOn,omitempty"`
	Location   string                 `json:"location,omitempty"`
	Tags       map[string]interface{} `json:"tags,omitempty"`
	Copy       *Copy                  `json:"copy,omitempty"`
	Comments   string                 `json:"comments,omitempty"`
}

// Copy represents an ARM template copy stanza
type Copy struct {
	Name      string `json:"name,omitempty"`
	Count     int    `json:"count,omitempty"`
	Mode      string `json:"mode,omitempty"`
	BatchSize int    `json:"batchSize,omitempty"`
}

// Output represents an ARM template output
type Output struct {
	Condition bool        `json:"condition,omitempty"`
	Type      string      `json:"type,omitempty"`
	Value     interface{} `json:"value,omitempty"`
}

// Parameters represents ARM parameters
type Parameters struct {
	Schema         string                          `json:"$schema,omitempty"`
	ContentVersion string                          `json:"contentVersion,omitempty"`
	Parameters     map[string]*ParametersParameter `json:"parameters,omitempty"`
}

// GetParametersMapInterface returns map[string]interface{} of the parameters field
// for ARM API to consume
func (p Parameters) GetParametersMapInterface() (map[string]interface{}, error) {
	var rawParameters map[string]interface{}
	data, err := json.Marshal(p.Parameters)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &rawParameters)
	if err != nil {
		return nil, err
	}
	return rawParameters, nil
}

// ParametersParameter represents an ARM parameters parameter
type ParametersParameter struct {
	Ref      string                 `json:"$ref,omitempty"`
	Value    interface{}            `json:"value,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
