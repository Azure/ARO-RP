package swagger

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/orderedmap"
)

// Swagger represents a Swagger object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#swagger-object
type Swagger struct {
	Swagger             string                 `json:"swagger,omitempty"`
	Info                *Info                  `json:"info,omitempty"`
	Host                string                 `json:"host,omitempty"`
	BasePath            string                 `json:"basePath,omitempty"`
	Schemes             []string               `json:"schemes,omitempty"`
	Consumes            []string               `json:"consumes,omitempty"`
	Produces            []string               `json:"produces,omitempty"`
	Paths               Paths                  `json:"paths,omitempty"`
	Definitions         Definitions            `json:"definitions,omitempty"`
	Parameters          ParametersDefinitions  `json:"parameters,omitempty"`
	Responses           ResponsesDefinitions   `json:"responses,omitempty"`
	SecurityDefinitions SecurityDefinitions    `json:"securityDefinitions,omitempty"`
	Security            []SecurityRequirement  `json:"security,omitempty"`
	Tags                []*Tag                 `json:"tags,omitempty"`
	ExternalDocs        *ExternalDocumentation `json:"externalDocs,omitempty"`
}

// Info represents an Info object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#info-object
type Info struct {
	Title          string   `json:"title,omitempty"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version,omitempty"`
}

// Contact represents a Contact object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#contact-object
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License represents a License object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#license-object
type License struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Paths represents a Paths object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#paths-object
type Paths map[string]*PathItem

// PathItem represents a Path Item object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#path-item-object
type PathItem struct {
	Ref        string        `json:"$ref,omitempty"`
	Get        *Operation    `json:"get,omitempty"`
	Put        *Operation    `json:"put,omitempty"`
	Post       *Operation    `json:"post,omitempty"`
	Delete     *Operation    `json:"delete,omitempty"`
	Options    *Operation    `json:"options,omitempty"`
	Head       *Operation    `json:"head,omitempty"`
	Patch      *Operation    `json:"patch,omitempty"`
	Parameters []interface{} `json:"parameters,omitempty"`
}

// Operation represents an Operation object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#operation-object
type Operation struct {
	Tags         []string               `json:"tags,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	OperationID  string                 `json:"operationId,omitempty"`
	Consumes     []string               `json:"consumes,omitempty"`
	Produces     []string               `json:"produces,omitempty"`
	Parameters   []interface{}          `json:"parameters,omitempty"`
	Responses    Responses              `json:"responses,omitempty"`
	Schemes      []string               `json:"schemes,omitempty"`
	Deprecated   bool                   `json:"deprecated,omitempty"`

	LongRunningOperation bool                 `json:"x-ms-long-running-operation,omitempty"`
	Examples             map[string]Reference `json:"x-ms-examples,omitempty"`
	Pageable             *Pageable            `json:"x-ms-pageable,omitempty"`
}

// ExternalDocumentation represents an External Documentation object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#external-documentation-object
type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// Pageable represents a Pageable object
type Pageable struct {
	NextLinkName string `json:"nextLinkName,omitempty"`
}

// Parameter represents a Parameter object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#parameter-object
type Parameter struct {
	Name             string        `json:"name,omitempty"`
	In               string        `json:"in,omitempty"`
	Description      string        `json:"description,omitempty"`
	Required         bool          `json:"required,omitempty"`
	Schema           *Schema       `json:"schema,omitempty"`
	Type             string        `json:"type,omitempty"`
	Format           string        `json:"format,omitempty"`
	AllowEmptyValue  bool          `json:"allowEmptyValue,omitempty"`
	Items            *Items        `json:"items,omitempty"`
	CollectionFormat string        `json:"collectionFormat,omitempty"`
	Default          interface{}   `json:"default,omitempty"`
	Maximum          int           `json:"maximum,omitempty"`
	ExclusiveMaximum bool          `json:"exclusiveMaximum,omitempty"`
	Minimum          int           `json:"minimum,omitempty"`
	ExclusiveMinimum bool          `json:"exclusiveMinimum,omitempty"`
	MaxLength        int           `json:"maxLength,omitempty"`
	MinLength        int           `json:"minLength,omitempty"`
	Pattern          string        `json:"pattern,omitempty"`
	MaxItems         int           `json:"maxItems,omitempty"`
	MinItems         int           `json:"minItems,omitempty"`
	UniqueItems      bool          `json:"uniqueItems,omitempty"`
	Enum             []interface{} `json:"enum,omitempty"`
	MultipleOf       int           `json:"multipleOf,omitempty"`

	XMSParameterLocation string `json:"x-ms-parameter-location,omitempty"`
}

// Items represents an Items object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#items-object
type Items struct {
	Type             string        `json:"type,omitempty"`
	Format           string        `json:"format,omitempty"`
	Items            *Items        `json:"items,omitempty"`
	CollectionFormat string        `json:"collectionFormat,omitempty"`
	Default          interface{}   `json:"default,omitempty"`
	Maximum          int           `json:"maximum,omitempty"`
	ExclusiveMaximum bool          `json:"exclusiveMaximum,omitempty"`
	Minimum          int           `json:"minimum,omitempty"`
	ExclusiveMinimum bool          `json:"exclusiveMinimum,omitempty"`
	MaxLength        int           `json:"maxLength,omitempty"`
	MinLength        int           `json:"minLength,omitempty"`
	Pattern          string        `json:"pattern,omitempty"`
	MaxItems         int           `json:"maxItems,omitempty"`
	MinItems         int           `json:"minItems,omitempty"`
	UniqueItems      bool          `json:"uniqueItems,omitempty"`
	Enum             []interface{} `json:"enum,omitempty"`
	MultipleOf       int           `json:"multipleOf,omitempty"`
}

// Responses represents a Responses object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#responses-object
type Responses map[string]interface{}

// Response represents a Response object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#response-object
type Response struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Headers     Headers `json:"headers,omitempty"`
	Examples    Example `json:"examples,omitempty"`
}

// Headers represents a Headers object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#headers-object
type Headers map[string]*Header

// Example represents an Example object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#example-object
type Example map[string]interface{}

// Header represents a Header object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#header-object
type Header struct {
	Description      string        `json:"description,omitempty"`
	Type             string        `json:"type,omitempty"`
	Format           string        `json:"format,omitempty"`
	Items            *Items        `json:"items,omitempty"`
	CollectionFormat string        `json:"collectionFormat,omitempty"`
	Default          interface{}   `json:"default,omitempty"`
	Maximum          int           `json:"maximum,omitempty"`
	ExclusiveMaximum bool          `json:"exclusiveMaximum,omitempty"`
	Minimum          int           `json:"minimum,omitempty"`
	ExclusiveMinimum bool          `json:"exclusiveMinimum,omitempty"`
	MaxLength        int           `json:"maxLength,omitempty"`
	MinLength        int           `json:"minLength,omitempty"`
	Pattern          string        `json:"pattern,omitempty"`
	MaxItems         int           `json:"maxItems,omitempty"`
	MinItems         int           `json:"minItems,omitempty"`
	UniqueItems      bool          `json:"uniqueItems,omitempty"`
	Enum             []interface{} `json:"enum,omitempty"`
	MultipleOf       int           `json:"multipleOf,omitempty"`
}

// Tag represents a Tag object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#tag-object
type Tag struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

// Reference represents a Reference object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#reference-object
type Reference struct {
	Ref string `json:"$ref,omitempty"`
}

// Schema represents a Schema object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#schema-object
type Schema struct {
	Ref                  string                 `json:"$ref,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	MultipleOf           int                    `json:"multipleOf,omitempty"`
	Maximum              int                    `json:"maximum,omitempty"`
	ExclusiveMaximum     bool                   `json:"exclusiveMaximum,omitempty"`
	Minimum              int                    `json:"minimum,omitempty"`
	ExclusiveMinimum     bool                   `json:"exclusiveMinimum,omitempty"`
	MaxLength            int                    `json:"maxLength,omitempty"`
	MinLength            int                    `json:"minLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	MaxItems             int                    `json:"maxItems,omitempty"`
	MinItems             int                    `json:"minItems,omitempty"`
	UniqueItems          bool                   `json:"uniqueItems,omitempty"`
	MaxProperties        int                    `json:"maxProperties,omitempty"`
	MinProperties        int                    `json:"minProperties,omitempty"`
	Required             bool                   `json:"required,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	Items                *Schema                `json:"items,omitempty"`
	AllOf                []Schema               `json:"allOf,omitempty"`
	Properties           NameSchemas            `json:"properties,omitempty"`
	AdditionalProperties *Schema                `json:"additionalProperties,omitempty"`
	Discriminator        string                 `json:"discriminator,omitempty"`
	ReadOnly             bool                   `json:"readOnly,omitempty"`
	XML                  *XML                   `json:"xml,omitempty"`
	ExternalDocs         *ExternalDocumentation `json:"externalDocs,omitempty"`
	Example              interface{}            `json:"example,omitempty"`

	ClientFlatten  bool      `json:"x-ms-client-flatten,omitempty"`
	XMSEnum        *XMSEnum  `json:"x-ms-enum,omitempty"`
	XMSSecret      bool      `json:"x-ms-secret,omitempty"`
	XMSIdentifiers *[]string `json:"x-ms-identifiers,omitempty"`
}

// XMSEnum is x-ms-enum swagger extension adding the ability to generate static enums
// https://github.com/Azure/autorest/tree/master/docs/extensions#x-ms-enum
type XMSEnum struct {
	Name          string         `json:"name"`
	ModelAsString bool           `json:"modelAsString"`
	Values        []XMSEnumValue `json:"values,omitempty"`
}

// XMSEnumValue represents value for x-ms-enum
// https://github.com/Azure/autorest/tree/master/docs/extensions#x-ms-enum
type XMSEnumValue struct {
	Value       interface{} `json:"value"`
	Description *string     `json:"description,omitempty"`
	Name        *string     `json:"name,omitempty"`
}

// XML represents an XML object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#xml-object
type XML struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// Definitions represents a Definitions object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#definitions-object
type Definitions map[string]*Schema

// ParametersDefinitions represents a Parameters Definitions object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#parameters-definitions-object
type ParametersDefinitions map[string]*Parameter

// ResponsesDefinitions represents a Responses Definitions object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#responses-definitions-object
type ResponsesDefinitions map[string]*Response

// SecurityDefinitions represents a Security Definitions object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#security-definitions-object
type SecurityDefinitions map[string]*SecurityScheme

// SecurityScheme represents a Security Scheme object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#security-scheme-object
type SecurityScheme struct {
	Type             string `json:"type,omitempty"`
	Description      string `json:"description,omitempty"`
	Name             string `json:"name,omitempty"`
	In               string `json:"in,omitempty"`
	Flow             string `json:"flow,omitempty"`
	AuthorizationURL string `json:"authorizationUrl,omitempty"`
	TokenURL         string `json:"tokenUrl,omitempty"`
	Scopes           Scopes `json:"scopes,omitempty"`
}

// Scopes represents a Scopes object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#scopes-object
type Scopes map[string]string

// SecurityRequirement represents a Security Requirement object
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#security-requirement-object
type SecurityRequirement map[string][]string

// from here downwards the types don't match the Swagger specification:
// NameSchemas uses orderedmap to ensure that the ordering of properties stanzas
// is respected.

// NameParameters is a slice of NameParameters
type NameParameters []NameParameter

// NameParameter represents a name and a Parameter
type NameParameter struct {
	Name      string
	Parameter interface{}
}

// UnmarshalJSON implements json.Unmarshaler
func (xs *NameParameters) UnmarshalJSON(b []byte) error {
	return orderedmap.UnmarshalJSON(b, xs)
}

// MarshalJSON implements json.Marshaler
func (xs NameParameters) MarshalJSON() ([]byte, error) {
	return orderedmap.MarshalJSON(xs)
}

// NameSchemas is a slice of NameSchemas
type NameSchemas []NameSchema

// NameSchema represents a name and a Schema
type NameSchema struct {
	Name   string
	Schema *Schema
}

// UnmarshalJSON implements json.Unmarshaler
func (xs *NameSchemas) UnmarshalJSON(b []byte) error {
	return orderedmap.UnmarshalJSON(b, xs)
}

// MarshalJSON implements json.Marshaler
func (xs NameSchemas) MarshalJSON() ([]byte, error) {
	return orderedmap.MarshalJSON(xs)
}
