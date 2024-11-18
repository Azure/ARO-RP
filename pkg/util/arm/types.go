package arm

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
	Resource interface{} `json:"-"`

	Name       string                 `json:"name,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Condition  interface{}            `json:"condition,omitempty"`
	APIVersion string                 `json:"apiVersion,omitempty"`
	DependsOn  []string               `json:"dependsOn,omitempty"`
	Location   string                 `json:"location,omitempty"`
	Tags       map[string]interface{} `json:"tags,omitempty"`
	Copy       *Copy                  `json:"copy,omitempty"`
	Comments   string                 `json:"comments,omitempty"`
	// Etag is required to omit Etag member when it's not set.
	// arm* SDK uses its own MarshalJSON, and empty members are converted to nil,
	// but Etag: nil fails the ARM template validation.
	Etag string `json:"etag,omitempty"`
}

// Deployment represents a nested ARM deployment in a deployment
type Deployment struct {
	Name       string                `json:"name,omitempty"`
	Type       string                `json:"type,omitempty"`
	Location   string                `json:"location,omitempty"`
	APIVersion string                `json:"apiVersion,omitempty"`
	DependsOn  []string              `json:"dependsOn,omitempty"`
	Condition  interface{}           `json:"condition,omitempty"`
	Properties *DeploymentProperties `json:"properties,omitempty"`
}

// DeploDeploymentProperties represents the propertioes of a nested ARM deployment
type DeploymentProperties struct {
	Mode                        string                                          `json:"mode,omitempty"`
	ExpressionEvaluationOptions map[string]*string                              `json:"expressionEvaluationOptions,omitempty"`
	Parameters                  map[string]*DeploymentTemplateResourceParameter `json:"parameters,omitempty"`
	Variables                   map[string]interface{}                          `json:"variables,omitempty"`
	Template                    *DeploymentTemplate                             `json:"template,omitempty"`
}

// DeploymentTemplate represents the inner template of a nested ARM deployment
type DeploymentTemplate struct {
	Schema         string                        `json:"$schema,omitempty"`
	APIProfile     string                        `json:"apiProfile,omitempty"`
	ContentVersion string                        `json:"contentVersion,omitempty"`
	Variables      map[string]interface{}        `json:"variables,omitempty"`
	Parameters     map[string]*TemplateParameter `json:"parameters,omitempty"`
	Functions      []interface{}                 `json:"functions,omitempty"`
	Resources      []*DeploymentTemplateResource `json:"resources,omitempty"`
}

// DeploymentTemplateResource represents the inner template's resource of a nested ARM deployment
type DeploymentTemplateResource struct {
	Name       string      `json:"name,omitempty"`
	Type       string      `json:"type,omitempty"`
	APIVersion string      `json:"apiVersion,omitempty"`
	Properties interface{} `json:"properties,omitempty"`
}

// DeploymentTemplateResourceParameter represents a nested ARM deployment's resource parameter
type DeploymentTemplateResourceParameter struct {
	Value string `json:"value,omitempty"`
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

// ParametersParameter represents an ARM parameters parameter
type ParametersParameter struct {
	Ref      string                 `json:"$ref,omitempty"`
	Value    interface{}            `json:"value,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
